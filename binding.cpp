#include <node.h>
#include <iostream>
#include <vector>

namespace murmur {

using v8::FunctionCallbackInfo;
using v8::Isolate;
using v8::Local;
using v8::Object;
using v8::String;
using v8::Value;
//-------------------


typedef std::vector<unsigned char> Buffer;

void print_usage();
Buffer get_jar_contents(const char* jar_file_path);
long get_file_size(FILE* file);
uint32_t compute_hash(Buffer& buffer);
bool is_whitespace_character(char b);
uint32_t compute_normalized_length(Buffer& buffer);

// -----------------------------------------------------------------------------
int hashIt(const FunctionCallbackInfo<Value>& args) {
  Isolate* isolate = args.GetIsolate();
  v8::String::Utf8Value str(isolate, args[0]);
  std::string cppStr(*str);

  const char* path = *str;

  Buffer jar_buffer = get_jar_contents(path);
  if (jar_buffer.empty()) {
    std::cout << "Failed to load " << path << std::endl;
    return 1;
  }

  const uint32_t hash = compute_hash(jar_buffer);
  args.GetReturnValue().Set(hash);
  return hash;
}

Buffer get_jar_contents(const char* jar_file_path) {
  FILE* jar_file = fopen(jar_file_path, "rb");
  if (jar_file == nullptr) {
    return Buffer();
  }

  long buffer_size = get_file_size(jar_file);

  Buffer buffer(buffer_size);
  fread(buffer.data(), 1, buffer_size, jar_file);
  fclose(jar_file);

  return buffer;
}

long get_file_size(FILE* file) {
  fseek(file, 0, SEEK_END);
  long size = ftell(file);
  fseek(file, 0, SEEK_SET);

  return size;
}

uint32_t compute_hash(Buffer& buffer) {
  const uint32_t multiplex = 1540483477;
  const uint32_t length = buffer.size();
  const uint32_t testI = 32;
  uint32_t num1 = length;

  num1 = compute_normalized_length(buffer);

  uint32_t num2 = (uint32_t)1 ^ num1;
  uint32_t num3 = 0;
  uint32_t num4 = 0;

  for (int index = 0; index < length; ++index) {

    unsigned char b = buffer[index];

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

    if (index == testI) {
      std::cout << "b: " << std::endl;
      std::cout << "num1: " << num1 << std::endl;
      std::cout << "num2: " << num2 << std::endl;
      std::cout << "num3: " << num3 << std::endl;
      std::cout << "num4: " << num4 << std::endl;
    }
  }

  if (num4 > 0) {
    num2 = (num2 ^ num3) * multiplex;
  }

  uint32_t num6 = (num2 ^ num2 >> 13) * multiplex;

  return num6 ^ num6 >> 15;
}

uint32_t compute_normalized_length(Buffer& buffer) {
  int32_t num1 = 0;
  const uint32_t length = buffer.size();

  for (int32_t index = 0; index < length; ++index) {
    if (!is_whitespace_character(buffer[index])) {
      ++num1;
    }
  }

  return num1;
}

bool is_whitespace_character(char b) {
  return b == 9 || b == 10 || b == 13 || b == 32;
}

//-------------------

void Initialize(Local<Object> exports) {
  NODE_SET_METHOD(exports, "hash", hashIt);
}

NODE_MODULE(NODE_GYP_MODULE_NAME, Initialize)

}
