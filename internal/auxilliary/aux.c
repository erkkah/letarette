#include "aux.h"
#include <string.h>
#include <stdlib.h>

static void firstmatch(
    const Fts5ExtensionApi *pApi,   // API offered by current FTS version
    Fts5Context *pFts,              // First arg to pass to pApi functions
    sqlite3_context *pCtx,          // Context for returning result/error
    int nVal,                       // Number of values in apVal[] array
    sqlite3_value **apVal           // Array of trailing arguments
) {
    int matchCount = 0;
    int result = pApi->xInstCount(pFts, &matchCount);
    if (result != SQLITE_OK) {
        sqlite3_result_error_code(pCtx, result);
        return;
    }

    int phrase = 0;
    int column = 0;
    int offset = 0;
    result = pApi->xInst(pFts, 0, &phrase, &column, &offset);
    if (result != SQLITE_OK) {
        sqlite3_result_error_code(pCtx, result);
        return;
    }
    sqlite3_result_int(pCtx, offset);
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

static void gettokens(
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

int initAuxilliaryFunctions(sqlite3* db) {
    fts5_api* fts = fts5APIFromDB(db);
    if (!fts) {
        return SQLITE_ERROR;
    }

    int result = fts->xCreateFunction(
        // firstmatch(fts)
        fts, "firstmatch", (void*) 0, firstmatch, (void*) 0
    );

    if (result != SQLITE_OK) {
        return result;
    }

    result = fts->xCreateFunction(
        // gettokens(fts, text, starttoken, count)
        fts, "gettokens", (void*) 0, gettokens, (void*) 0
    );

    return result;
}
