cmake_minimum_required(VERSION 3.14)

project(AirlinkStreamBridge LANGUAGES C CXX VERSION 1.0.0)

set(CMAKE_CXX_STANDARD 17)
set(SIMDJSON_SINGLE_HEADER_DIR external/libs/simdjson/singleheader)
set(LIBSDPTRANSFORM_DIR external/libs/libsdptransform)


if ("${CMAKE_CXX_COMPILER_ID}" STREQUAL "Clang" OR "${CMAKE_CXX_COMPILER_ID}" STREQUAL "GNU")
    add_compile_options(-Wall -Wextra)
    add_compile_options(-Wno-format-truncation)
endif()

file(GLOB SUBPROJECT_PATHS LIST_DIRECTORIES true "external/libs/*")
foreach(SUBPROJECT_PATH ${SUBPROJECT_PATHS})
    if(IS_DIRECTORY "${SUBPROJECT_PATH}" AND EXISTS "${SUBPROJECT_PATH}/CMakeLists.txt")
        add_subdirectory("${SUBPROJECT_PATH}")
    endif()
endforeach()

find_package(Qt6 COMPONENTS REQUIRED Network)

set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)

file(GLOB_RECURSE WEBRTC_TO_RTP_SOURCES src/*.cpp)
file(GLOB_RECURSE WEBRTC_TO_RTP_HEADERS src/*.h)


add_executable(${PROJECT_NAME})

target_compile_definitions(${PROJECT_NAME} PRIVATE WEBRTC_POSIX SIMDJSON_HEADER_ONLY)

file(GLOB LIB_SDP_TRANSFORM_SOURCES ${LIBSDPTRANSFORM_DIR}/src/*.cpp)
target_sources(${PROJECT_NAME} PRIVATE ${WEBRTC_TO_RTP_HEADERS} ${WEBRTC_TO_RTP_SOURCES} ${SIMDJSON_SINGLE_HEADER_DIR}/simdjson.cpp ${LIB_SDP_TRANSFORM_SOURCES})

target_link_libraries(${PROJECT_NAME} PRIVATE uvgrtp datachannel cpprest CLI11::CLI11)
target_link_libraries(${PROJECT_NAME} PRIVATE Qt6::Network)



target_include_directories(${PROJECT_NAME} PRIVATE ${SIMDJSON_SINGLE_HEADER_DIR} ${LIBSDPTRANSFORM_DIR}/include)