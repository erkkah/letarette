/*
 * Copyright 2019 Erik Agsjö
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "snowball.h"
#include <libstemmer.h>
#include <string.h>
#include <stdlib.h>

#if SQLITE_VERSION_NUMBER < 3024000
#error "Need at least SQLite 3.24.0."
#pragma message "Found SQLite " SQLITE_VERSION
#endif

#define MAX_TOKEN_LEN 40
#define MIN_TOKEN_LEN 3

struct StemmerModuleData {
    sqlite3 *db;
    struct sb_stemmer** stemmers;
    int minTokenLength;
    const char** parentArgs;
    int nParentArgs;
    fts5_api *fts;
};

struct StemmerInstance {
    struct StemmerModuleData* module;
	fts5_tokenizer parentModule;
	Fts5Tokenizer *parentInstance;
    sqlite3_stmt *stopWordStatement;
};

struct StemmerContext {
    struct StemmerInstance* instance;
    void* callerContext;
    int removeStopwords;
    int (*xToken)(void*, int, const char*, int, int, int);
};

static int ftsSnowballCreate(
	void *pCtx,
	const char **azArg, int nArg,
	Fts5Tokenizer **ppOut
){
    struct StemmerModuleData* modData = (struct StemmerModuleData*) pCtx;
    struct StemmerInstance* instance = (struct StemmerInstance*) sqlite3_malloc(sizeof(struct StemmerInstance));
    if (!instance) {
        return SQLITE_ERROR;
    }

    instance->module = modData;

    const char * const parentStemmer = "unicode61";
    void* parentUserData = 0;
    int rc = modData->fts->xFindTokenizer(modData->fts, parentStemmer, &parentUserData, &instance->parentModule);

    if (rc == SQLITE_OK) {
        rc = instance->parentModule.xCreate(parentUserData, modData->parentArgs, modData->nParentArgs, &instance->parentInstance);
    }

    instance->stopWordStatement = 0;

    if (rc == SQLITE_OK) {
        *ppOut = (Fts5Tokenizer*) instance;
    } else {
        sqlite3_free(instance);
    }

    return rc;
}

static void ftsSnowballDelete(Fts5Tokenizer *pTok) {
    struct StemmerInstance* instance = (struct StemmerInstance*) pTok;
    instance->parentModule.xDelete(instance->parentInstance);
    sqlite3_free(instance);
}

static int isStopWord(struct StemmerInstance* instance, const char* word, int len) {
    sqlite3_stmt *s = instance->stopWordStatement;

    // Lazy init since stemmer is created before migrations are run
    if (s == 0) {
        static const char* const stopwordCheck = "select count(*) from stopwords where word=?";
        int rc = sqlite3_prepare_v2(instance->module->db, stopwordCheck, -1, &instance->stopWordStatement, 0);
        if (rc != SQLITE_OK) {
            return -1;
        }
        s = instance->stopWordStatement;
    }

    int rc = sqlite3_bind_text(s, 1, word, len, 0);
    if (rc != SQLITE_OK) {
        return -2;
    }
    rc = sqlite3_step(s);
    if (rc != SQLITE_ROW) {
        return -3;
    }
    int exists = sqlite3_column_int(s, 0);
    rc = sqlite3_reset(s);
    if (rc != SQLITE_OK) {
        return -4;
    }
    return exists;
}

static int ftsSnowballCallback(
	void *pCtx,
	int tflags,
	const char *pToken,
	int nToken,
	int iStart,
	int iEnd
){
    struct StemmerContext* ctx = (struct StemmerContext*) pCtx;

    // Skip tokens below minTokenLength
    if (nToken < ctx->instance->module->minTokenLength) {
        return SQLITE_OK;
    }

    if (ctx->removeStopwords) {
        
        int stopwordStatus = isStopWord(ctx->instance, pToken, nToken);
        if (stopwordStatus < 0) {
            return SQLITE_ERROR;
        }

        if (stopwordStatus != 0) {
            return SQLITE_OK;
        }
    }

    // Only call snowball for tokens withing the set interval
    if (nToken > MAX_TOKEN_LEN || nToken < MIN_TOKEN_LEN) {
        return ctx->xToken(ctx->callerContext, tflags, pToken, nToken, iStart, iEnd);
    }

    char buffer[MAX_TOKEN_LEN];
    memcpy(buffer, pToken, nToken);
    struct sb_stemmer** stemmer = ctx->instance->module->stemmers;
    const sb_symbol* stemmed = (const sb_symbol*) pToken;
    int stemmedLength = nToken;
    while (*stemmer) {
        stemmed = sb_stemmer_stem(*stemmer, (unsigned char*) buffer, nToken);
        stemmedLength = sb_stemmer_length(*stemmer);
        if (stemmedLength != nToken) {
            break;
        }
        stemmer++;
    }
    return ctx->xToken(ctx->callerContext, tflags, (const char*) stemmed, stemmedLength, iStart, iEnd);
}

static int ftsSnowballTokenize(
	Fts5Tokenizer *pTokenizer,
	void *pCtx,
	int flags,
	const char *pText, int nText,
	int (*xToken)(void*, int, const char*, int nToken, int iStart, int iEnd)
){
    struct StemmerInstance* instance = (struct StemmerInstance*) pTokenizer;
    struct StemmerContext ctx;
    ctx.callerContext = pCtx;
    ctx.instance = instance;
    ctx.xToken = xToken;

    if ( (flags & (FTS5_TOKENIZE_QUERY | FTS5_TOKENIZE_PREFIX)) == FTS5_TOKENIZE_QUERY ) {
        ctx.removeStopwords = 1;

        for (int i = 0; i < nText; i++) {
            // No stop word handling for quoted phrases
            if (pText[i] == ' ') {
                ctx.removeStopwords = 0;
                break;
            }
        }
    } else {
        ctx.removeStopwords = 0;
    }

    return instance->parentModule.xTokenize(
        instance->parentInstance, &ctx, flags, pText, nText, ftsSnowballCallback
    );
}

static fts5_api *fts5APIFromDB(sqlite3 *db){
	fts5_api *pRet = 0;
	sqlite3_stmt *pStmt = 0;

	if (sqlite3_prepare(db, "SELECT fts5(?1)", -1, &pStmt, 0) == SQLITE_OK){
		sqlite3_bind_pointer(pStmt, 1, (void*)&pRet, "fts5_api_ptr", 0);
		sqlite3_step(pStmt);
	}
	sqlite3_finalize(pStmt);
	return pRet;
}

static void freeStemmerList(struct sb_stemmer** stemmers) {
    struct sb_stemmer** stemmer = stemmers;
    while(*stemmer) {
        sb_stemmer_delete(*stemmer);
        stemmer++;
    }
    sqlite3_free(stemmers);
}

static struct sb_stemmer** allocateStemmerList(const char** languages, int nLanguages) {
    struct sb_stemmer** stemmers = sqlite3_malloc((nLanguages + 1) * sizeof(struct sb_stemmer*));
    if (!stemmers) {
        return 0;
    }
    for (int i = 0; i < nLanguages; i++) {
        struct sb_stemmer* stemmer = sb_stemmer_new(languages[i], "UTF_8");
        stemmers[i] = stemmer;
        stemmers[i + 1] = 0;
        if (!stemmer) {
            freeStemmerList(stemmers);
            return 0;
        }
    }
    return stemmers;
}

static void destroyStemmerModule(void *p) {
    struct StemmerModuleData* modData = (struct StemmerModuleData*) p;
    freeStemmerList(modData->stemmers);
    for (int i = 0; i < modData->nParentArgs; i++) {
        sqlite3_free((void*) modData->parentArgs[i]);
    }
    sqlite3_free(modData->parentArgs);
    sqlite3_free(modData);
}

int initSnowballStemmer(
    sqlite3 *db,
    const char** languages,
    int nLanguages,
    int removeDiacritics,
    const char* tokenCharacters,
    const char* separators,
    int minTokenLength
){
    fts5_tokenizer tokenizer = {ftsSnowballCreate, ftsSnowballDelete, ftsSnowballTokenize};

    struct StemmerModuleData* modData = sqlite3_malloc(sizeof(struct StemmerModuleData));
    if (!modData) {
        return SQLITE_ERROR;
    }
    modData->db = db;

    struct sb_stemmer** stemmers = allocateStemmerList(languages, nLanguages);
    if (!stemmers) {
        sqlite3_free(modData);
        return SQLITE_ERROR;
    }

    modData->stemmers = stemmers;
    modData->minTokenLength = minTokenLength;

    const int maxArgs = 6;
    const char** args = sqlite3_malloc(sizeof(char*) * maxArgs);
    int nArgs = 0;

    args[nArgs++] = sqlite3_mprintf("remove_diacritics");
    args[nArgs++] = sqlite3_mprintf("%d", removeDiacritics);
    if (tokenCharacters) {
        args[nArgs++] = sqlite3_mprintf("tokenchars");
        args[nArgs++] = sqlite3_mprintf("'%s'", tokenCharacters);
    }
    if (separators) {
        args[nArgs++] = sqlite3_mprintf("separators");
        args[nArgs++] = sqlite3_mprintf("'%s'", separators);
    }

    modData->parentArgs = args;
    modData->nParentArgs = nArgs;

    modData->fts = fts5APIFromDB(db);
    if (!modData->fts) {
        return SQLITE_ERROR;
    }

    int result = modData->fts->xCreateTokenizer(
        modData->fts, "snowball", (void *) modData, &tokenizer, destroyStemmerModule
    );

    return result;
}

const char** getStemmerList() {
    return sb_stemmer_list();
}
