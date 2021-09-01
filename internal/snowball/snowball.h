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

#include <sqlite3-binding.h>

int initSnowballStemmer(
    sqlite3* db,
    const char** languages,
    int nLanguages,
    int removeDiacritics,
    const char* tokenCharacters,
    const char* separators,
    int minTokenLength
);

const char** getStemmerList();
