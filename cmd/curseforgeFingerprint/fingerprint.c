/**
This is a C port of the C++ code that Curseforge uses to create fingerprints.
Why they're not using a platform independent hashing algorithm is beyond me.

The original C++ code that I got from their support team
can be found here: https://github.com/meza/curseforge-fingerprint/blob/main/src/addon/fingerprint.cpp

The C++ -> C conversion is done via GitHub Copilot and some manual adjustments as I am not a C programmer.
*/

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    unsigned char* data;
    size_t size;
} Buffer;

Buffer get_jar_contents(const char* jar_file_path);
long get_file_size(FILE* file);
uint32_t compute_hash(const char* jar_file_path);
bool is_whitespace_character(char b);
uint32_t compute_normalized_length(Buffer buffer);

Buffer get_jar_contents(const char* jar_file_path) {
    Buffer buffer = {NULL, 0};
    FILE* jar_file = fopen(jar_file_path, "rb");
    if (jar_file == NULL) {
        return buffer;
    }

    long buffer_size = get_file_size(jar_file);
    buffer.data = (unsigned char*)malloc(buffer_size);
    if (buffer.data == NULL) {
        fclose(jar_file);
        return buffer;
    }
    buffer.size = buffer_size;

    size_t result = fread(buffer.data, 1, buffer_size, jar_file);
    if (result != buffer_size) {
        printf("Failed to load %s\n", jar_file_path);
        free(buffer.data);
        buffer.data = NULL;
        buffer.size = 0;
    }

    fclose(jar_file);
    return buffer;
}

long get_file_size(FILE* file) {
    fseek(file, 0, SEEK_END);
    long size = ftell(file);
    fseek(file, 0, SEEK_SET);
    return size;
}

uint32_t compute_hash(const char* jar_file_path) {
    Buffer buffer = get_jar_contents(jar_file_path);
    if (buffer.data == NULL) {
        return 0;
    }

    const uint32_t multiplex = 1540483477;
    const uint32_t length = buffer.size;
    uint32_t num1 = length;

    num1 = compute_normalized_length(buffer);

    uint32_t num2 = (uint32_t)1 ^ num1;
    uint32_t num3 = 0;
    uint32_t num4 = 0;

    for (uint32_t index = 0; index < length; ++index) {
        unsigned char b = buffer.data[index];
        if (!is_whitespace_character(b)) {
            num3 |= (uint32_t)b << num4;
            num4 += 8;
            if (num4 == 32) {
                uint32_t num6 = num3 * multiplex;
                uint32_t num7 = (num6 ^ num6 >> 24) * multiplex;
                num2 = num2 * multiplex ^ num7;
                num3 = 0;
                num4 = 0;
            }
        }
    }

    if (num4 > 0) {
        num2 = (num2 ^ num3) * multiplex;
    }

    uint32_t num6 = (num2 ^ num2 >> 13) * multiplex;
    free(buffer.data);
    return num6 ^ num6 >> 15;
}

uint32_t compute_normalized_length(Buffer buffer) {
    if (buffer.data == NULL) {
        return 0;
    }

    int32_t num1 = 0;
    const uint32_t length = buffer.size;

    for (uint32_t index = 0; index < length; ++index) {
        if (!is_whitespace_character(buffer.data[index])) {
            ++num1;
        }
    }

    return num1;
}

bool is_whitespace_character(char b) {
    return b == 9 || b == 10 || b == 13 || b == 32;
}
