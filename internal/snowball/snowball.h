#include <sqlite3.h>

int initSnowballStemmer(
    sqlite3* db,
    const char** languages,
    int nLanguages,
    int removeDiacritics,
    const char* tokenCharacters,
    const char* separators,
    int minTokenLength
);
