/*
** 2014-06-13
**
** The author disclaims copyright to this source code.  In place of
** a legal notice, here is a blessing:
**
**    May you do good and not evil.
**    May you find forgiveness for yourself and forgive others.
**    May you share freely, never taking more than you give.
**
******************************************************************************
**
** This SQLite extension implements SQL compression functions
** compress() and uncompress() using ZLIB.
**
** 2020-01-08
** Letarette-specific additions:
**
**   *   the "compress(X)" function handles errors by returning error
**
**   *   the "uncompress(X)" function handles errors by returning error, or the input if it is not in zlib format
**
**   *   Added "iscompressed(X)" that checks if the input is a compressed stream as returned from "compress(X)"
**
*/
#include "sqlite3ext.h"
SQLITE_EXTENSION_INIT1
#include "miniz.h"

/*
** Implementation of the "compress(X)" SQL function.  The input X is
** compressed using zLib and the output is returned.
**
** The output is a BLOB that begins with a one byte "magic" character
** that is not part of UTF8. This makes it faster to detect uncompressed
** data that would otherwise need to be passed to zlib.
** 
** After the macic byte, there is a variable-length integer that
** is the input size in bytes (the size of X before compression).  The
** variable-length integer is implemented as 1 to 5 bytes.  There are
** seven bits per integer stored in the lower seven bits of each byte.
** More significant bits occur first.  The most significant bit (0x80)
** is a flag to indicate the end of the integer.
**
** This function, SQLAR, and ZIP all use the same "deflate" compression
** algorithm, but each is subtly different:
**
**   *  ZIP uses raw deflate.
**
**   *  SQLAR uses the "zlib format" which is raw deflate with a two-byte
**      algorithm-identification header and a four-byte checksum at the end.
**
**   *  This utility uses the "zlib format" like SQLAR, but adds the variable-
**      length integer uncompressed size value at the beginning.
**
** This function might be extended in the future to support compression
** formats other than deflate, by providing a different algorithm-id
** mark following the variable-length integer size parameter.
*/
static void compressFunc(
  sqlite3_context *context,
  int argc,
  sqlite3_value **argv
){
  const unsigned char *pIn;
  unsigned char *pOut;
  unsigned int nIn;
  unsigned long int nOut;
  unsigned char x[8];
  int rc;
  int i, j;

  pIn = sqlite3_value_blob(argv[0]);
  nIn = sqlite3_value_bytes(argv[0]);
  nOut = 13 + nIn + (nIn+999)/1000;
  pOut = sqlite3_malloc( nOut+1+5 );

  for(i=4; i>=0; i--){
    x[i] = (nIn >> (7*(4-i)))&0x7f;
  }

  for(i=0; i<4 && x[i]==0; i++){}

  pOut[0] = 0xf8;
  for(j=1; i<=4; i++, j++) pOut[j] = x[i];
  pOut[j-1] |= 0x80;

  rc = compress(&pOut[j], &nOut, pIn, nIn);
  if( rc==Z_OK ){
    sqlite3_result_blob(context, pOut, nOut+j, sqlite3_free);
  }else{
    sqlite3_free(pOut);
    sqlite3_result_error_code(context, SQLITE_ERROR);
  }
}

static int isValidHeader(const char* data) {
  return
    (data[0] == 0x78 &&
    ((data[0] * 256) + data[1]) % 31 == 0);
}

/*
** Implementation of the "uncompress(X)" SQL function.  The argument X
** is a blob which was obtained from compress(Y).  The output will be
** the value Y.
**
** If the input is not a compressed stream as returned by "compress(X)",
** the unmodified input is returned.
*/
static void uncompressFunc(
  sqlite3_context *context,
  int argc,
  sqlite3_value **argv
){
  const unsigned char *pIn;
  unsigned char *pOut;
  unsigned int nIn;
  unsigned long int nOut;
  int rc;
  int i;

  pIn = sqlite3_value_blob(argv[0]);
  nIn = sqlite3_value_bytes(argv[0]);
  nOut = 0;
  if (pIn[0] == 0xf8) {
    for(i=1; i<nIn && i<6; i++){
      nOut = (nOut<<7) | (pIn[i]&0x7f);
      if( (pIn[i]&0x80)!=0 ){ i++; break; }
    }
  }

  if ((nOut == 0) || !isValidHeader(&pIn[i])) {
    sqlite3_result_blob(context, pIn, nIn, 0);
    return;
  }

  pOut = sqlite3_malloc( nOut+1 );
  rc = uncompress(pOut, &nOut, &pIn[i], nIn-i);
  if( rc == Z_OK ){
    sqlite3_result_blob(context, pOut, nOut, sqlite3_free);
  }else{
    sqlite3_free(pOut);
    // -10000 is MZ_PARAM_ERROR
    if ( rc == Z_DATA_ERROR || rc == Z_STREAM_ERROR || rc == -10000) {
      sqlite3_result_blob(context, pIn, nIn, 0);
    } else {
      sqlite3_result_error_code(context, SQLITE_ERROR);
    }
  }
}

/*
** Implementation of the "iscompressed(X)" SQL function. If the argument X
** is a blob which was obtained from compress(Y), the output will be true.
*/
static void isCompressedFunc(
  sqlite3_context *context,
  int argc,
  sqlite3_value **argv
){
  const unsigned char *pIn;
  unsigned char *pOut;
  unsigned int nIn;
  unsigned long int nOut;
  int rc;
  int i;

  pIn = sqlite3_value_blob(argv[0]);
  nIn = sqlite3_value_bytes(argv[0]);
  nOut = 0;
  if (pIn[0] == 0xf8) {
    for(i=1; i<nIn && i<6; i++){
      nOut = (nOut<<7) | (pIn[i]&0x7f);
      if( (pIn[i]&0x80)!=0 ){ i++; break; }
    }
  }

  sqlite3_result_int(context, (nOut != 0) && isValidHeader(&pIn[i]));
}


#ifdef _WIN32
__declspec(dllexport)
#endif
int sqlite3_compress_init(
  sqlite3 *db, 
  char **pzErrMsg, 
  const sqlite3_api_routines *pApi
){
  int rc = SQLITE_OK;
  SQLITE_EXTENSION_INIT2(pApi);
  (void)pzErrMsg;  /* Unused parameter */
  rc = sqlite3_create_function(db, "compress", 1, SQLITE_UTF8, 0,
                               compressFunc, 0, 0);
  if( rc != SQLITE_OK ){
    return rc;
  }
  rc = sqlite3_create_function(db, "uncompress", 1, SQLITE_UTF8, 0,
                               uncompressFunc, 0, 0);
  if( rc != SQLITE_OK ){
    return rc;
  }
  rc = sqlite3_create_function(db, "iscompressed", 1, SQLITE_UTF8, 0,
                               isCompressedFunc, 0, 0);
  if( rc != SQLITE_OK ){
    return rc;
  }
  return rc;
}
