#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    unsigned char* data;
    size_t size;
} Buffer;

void print_usage();
Buffer get_jar_contents(const char* jar_file_path);
long get_file_size(FILE* file);
uint32_t compute_hash(const char* jar_file_path);
bool is_whitespace_character(char b);
uint32_t compute_normalized_length(Buffer buffer);
