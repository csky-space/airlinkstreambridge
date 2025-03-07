cmake_minimum_required(VERSION 3.14)

project(WebRTCtoRtp LANGUAGES C CXX VERSION 1.0.0)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_PREFIX_PATH "/home/szamaro/Qt/6.8.2/gcc_64/lib/cmake")

set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)



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

find_package(Qt6 COMPONENTS REQUIRED Core Network)

file(GLOB_RECURSE WEBRTC_TO_RTP_SOURCES src/*.cpp)
file(GLOB_RECURSE WEBRTC_TO_RTP_HEADERS src/*.h)


add_executable(WebRTCtoRtp)

target_compile_definitions(WebRTCtoRtp PRIVATE WEBRTC_POSIX SIMDJSON_HEADER_ONLY)

file(GLOB LIB_SDP_TRANSFORM_SOURCES external/libs/libsdptransform/src/*.cpp)
target_sources(WebRTCtoRtp PRIVATE ${WEBRTC_TO_RTP_HEADERS} ${WEBRTC_TO_RTP_SOURCES} external/libs/simdjson/singleheader/simdjson.cpp ${LIB_SDP_TRANSFORM_SOURCES})

target_link_libraries(WebRTCtoRtp PRIVATE uvgrtp datachannel cpprest)
target_link_libraries(WebRTCtoRtp PRIVATE Qt6::Core Qt6::Network)



target_include_directories(WebRTCtoRtp PRIVATE /usr/local/include external/libs/simdjson/singleheader external/libs/libsdptransform/include)