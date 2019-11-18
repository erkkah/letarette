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

#include "aux.h"
#include <string.h>
#include <stdlib.h>

struct MatchData {
    sqlite3_int64 rowid;
    int phrase;
    int column;
    int offset;
};

static void firstMatch(
    const Fts5ExtensionApi *pApi,   // API offered by current FTS version
    Fts5Context *pFts,              // First arg to pass to pApi functions
    sqlite3_context *pCtx,          // Context for returning result/error
    int nVal,                       // Number of values in apVal[] array
    sqlite3_value **apVal           // Array of trailing arguments
) {
    if (nVal != 1) {
        sqlite3_result_error_code(pCtx, SQLITE_ERROR);
        return;
    }
    int columnOrOffset = sqlite3_value_int(apVal[0]);

    sqlite3_int64 rowid = pApi->xRowid(pFts);

    struct MatchData* cached = pApi->xGetAuxdata(pFts, 0);
    if (cached != 0 && cached->rowid == rowid) {
        sqlite3_result_int(pCtx, columnOrOffset ? cached->offset : cached->column);
        return;
    }

    int phrase = 0;
    int column = 0;
    int offset = 0;
    int result = pApi->xInst(pFts, 0, &phrase, &column, &offset);
    if (result != SQLITE_OK) {
        sqlite3_result_error_code(pCtx, result);
        return;
    }

    if (cached == 0) {
        cached = sqlite3_malloc(sizeof(struct MatchData));
        result = pApi->xSetAuxdata(pFts, cached, sqlite3_free);
        if (result != SQLITE_OK) {
            sqlite3_result_error_code(pCtx, result);
            return;
        }
    }
    cached->rowid = rowid;
    cached->phrase = phrase;
    cached->column = column;
    cached->offset = offset;

    sqlite3_result_int(pCtx, columnOrOffset ? offset : column);
}

struct TokenRangeContext {
    int tokenStart;
    int tokenEnd;
    int currentToken;
    int textRangeStart;
    int textRangeEnd;
};

static int tokenRangeCallback(
    void *pCtx,
    int tflags,
    const char *pToken,
    int nToken,
    int iStart,
    int iEnd
) {
    struct TokenRangeContext* ctx = (struct TokenRangeContext*) pCtx;

    if (ctx->currentToken == ctx->tokenStart) {
        ctx->textRangeStart = iStart;
    }

    if (ctx->currentToken >= ctx->tokenStart) {
        ctx->textRangeEnd = iEnd;
    }

    ctx->currentToken++;

    if (ctx->currentToken == ctx->tokenEnd) {
        return SQLITE_DONE;
    } else {
        return SQLITE_OK;
    }
}

static void getTokens(
    const Fts5ExtensionApi *pApi,   // API offered by current FTS version
    Fts5Context *pFts,              // First arg to pass to pApi functions
    sqlite3_context *pCtx,          // Context for returning result/error
    int nVal,                       // Number of values in apVal[] array
    sqlite3_value **apVal           // Array of trailing arguments
) {
    if (nVal != 3) {
        sqlite3_result_error_code(pCtx, SQLITE_ERROR);
        return;
    }
    const char* entry = (const char*) sqlite3_value_text(apVal[0]);
    int length = sqlite3_value_bytes(apVal[0]);
    int offset = sqlite3_value_int(apVal[1]);
    int count = sqlite3_value_int(apVal[2]);

    if (count < 0 || offset < 0) {
        sqlite3_result_error_code(pCtx, SQLITE_ERROR);
        return;
    }

    struct TokenRangeContext ctx = {
        offset,offset + count,0,0,0
    };

    int result = pApi->xTokenize(pFts, entry, length, &ctx, tokenRangeCallback);
    if (result != SQLITE_OK && result != SQLITE_DONE) {
        sqlite3_result_error_code(pCtx, result);
        return;
    }

    const char* snippetStart = entry + ctx.textRangeStart;
    int snippetLength = strnlen(snippetStart, ctx.textRangeEnd - ctx.textRangeStart);
    const char* snippet = strndup(snippetStart, snippetLength);
    sqlite3_result_text(pCtx, snippet, snippetLength, free);
}

static void tokenCount(
    const Fts5ExtensionApi *pApi,   // API offered by current FTS version
    Fts5Context *pFts,              // First arg to pass to pApi functions
    sqlite3_context *pCtx,          // Context for returning result/error
    int nVal,                       // Number of values in apVal[] array
    sqlite3_value **apVal           // Array of trailing arguments
) {
    if (nVal != 1) {
        sqlite3_result_error_code(pCtx, SQLITE_ERROR);
        return;
    }
    int column = sqlite3_value_int(apVal[0]);
    int tokens = 0;
    int result = pApi->xColumnSize(pFts, column, &tokens);
    if (result != SQLITE_OK) {
        sqlite3_result_error_code(pCtx, result);
        return;
    }
    sqlite3_result_int(pCtx, tokens);
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

int initAuxiliaryFunctions(sqlite3* db) {
    fts5_api* fts = fts5APIFromDB(db);
    if (!fts) {
        return SQLITE_ERROR;
    }

    int result = fts->xCreateFunction(
        // firstmatch(fts)
        fts, "firstmatch", (void*) 0, firstMatch, (void*) 0
    );

    if (result != SQLITE_OK) {
        return result;
    }

    result = fts->xCreateFunction(
        // gettokens(fts, text, starttoken, count)
        fts, "gettokens", (void*) 0, getTokens, (void*) 0
    );

    if (result != SQLITE_OK) {
        return result;
    }

    result = fts->xCreateFunction(
        // tokens(fts)
        fts, "tokens", (void*) 0, tokenCount, (void*) 0
    );

    return result;
}
