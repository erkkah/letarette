#include "snowball.h"
#include <libstemmer.h>
#include <string.h>
#include <stdlib.h>

#define MAX_TOKEN_LEN 64
#define MIN_TOKEN_LEN 3

struct StemmerModuleData {
    struct sb_stemmer** stemmers;
    const char** parentArgs;
    int nParentArgs;
    fts5_api *fts;
};

struct StemmerInstance {
    struct StemmerModuleData* module;
	fts5_tokenizer parentModule;
	Fts5Tokenizer *parentInstance;
};

struct StemmerContext {
    struct StemmerInstance* instance;
    void* callerContext;
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

    const char const* parentStemmer = "unicode61";
    void* parentUserData = 0;
    int rc = modData->fts->xFindTokenizer(modData->fts, parentStemmer, &parentUserData, &instance->parentModule);

    if (rc == SQLITE_OK) {
        rc = instance->parentModule.xCreate(parentUserData, modData->parentArgs, modData->nParentArgs, &instance->parentInstance);
    }

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

static int ftsSnowballCallback(
	void *pCtx,
	int tflags,
	const char *pToken,
	int nToken,
	int iStart,
	int iEnd
){
    struct StemmerContext* ctx = (struct StemmerContext*) pCtx;

    if (nToken > MAX_TOKEN_LEN || nToken < MIN_TOKEN_LEN) {
        return ctx->xToken(ctx->callerContext, tflags, pToken, nToken, iStart, iEnd);
    } else {
        char buffer[MAX_TOKEN_LEN];
        memcpy(buffer, pToken, nToken);
        struct sb_stemmer** stemmer = ctx->instance->module->stemmers;
        const sb_symbol* stemmed = pToken;
        int stemmedLength = nToken;
        while (*stemmer) {
            stemmed = sb_stemmer_stem(*stemmer, (unsigned char*) buffer, nToken);
            stemmedLength = sb_stemmer_length(*stemmer);
            if (stemmedLength != nToken) {
                break;
            }
            stemmer++;
        }
        return ctx->xToken(ctx->callerContext, tflags, stemmed, stemmedLength, iStart, iEnd);
    }
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

    instance->parentModule.xTokenize(
        instance->parentInstance, &ctx, flags, pText, nText, ftsSnowballCallback
    );
}

static fts5_api *fts5_api_from_db(sqlite3 *db){
	fts5_api *pRet = 0;
	sqlite3_stmt *pStmt = 0;

	if( SQLITE_OK==sqlite3_prepare(db, "SELECT fts5(?1)", -1, &pStmt, 0) ){
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
    const char* separators
){
    fts5_tokenizer tokenizer = {ftsSnowballCreate, ftsSnowballDelete, ftsSnowballTokenize};

    struct StemmerModuleData* modData = sqlite3_malloc(sizeof(struct StemmerModuleData));
    if (!modData) {
        return SQLITE_ERROR;
    }

    struct sb_stemmer** stemmers = allocateStemmerList(languages, nLanguages);
    if (!stemmers) {
        sqlite3_free(modData);
        return SQLITE_ERROR;
    }

    modData->stemmers = stemmers;

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

    modData->fts = fts5_api_from_db(db);

    int result = modData->fts->xCreateTokenizer(
        modData->fts, "snowball", (void *) modData, &tokenizer, destroyStemmerModule
    );

    return result;
}
