cmake_minimum_required(VERSION 3.14)
cmake_policy(SET CMP0100 NEW)

project(AirlinkStreamBridge LANGUAGES C CXX VERSION 1.0.0)

set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_CXX_STANDARD 17)

set(LIBSDPTRANSFORM_DIR ${CMAKE_CURRENT_SOURCE_DIR}/external/libs/libsdptransform)


if ("${CMAKE_CXX_COMPILER_ID}" STREQUAL "Clang" OR "${CMAKE_CXX_COMPILER_ID}" STREQUAL "GNU")
    add_compile_options(-Wall -Wextra)
    add_compile_options(-Wno-format-truncation)
endif()

set(UVGRTP_DISABLE_EXAMPLES ON)
set(UVGRTP_DISABLE_TESTS ON)
set(CLI11_BUILD_TESTS OFF)
set(NO_EXAMPLES ON)
set(NO_TESTS ON)

file(GLOB SUBPROJECT_PATHS LIST_DIRECTORIES true "${CMAKE_CURRENT_SOURCE_DIR}/external/libs/*")
foreach(SUBPROJECT_PATH ${SUBPROJECT_PATHS})
    if(IS_DIRECTORY "${SUBPROJECT_PATH}" AND EXISTS "${SUBPROJECT_PATH}/CMakeLists.txt")
        set(IS_TOP_LEVEL_VERSION_NEEDED ((${SUBPROJECT_PATH} STREQUAL "${CMAKE_CURRENT_SOURCE_DIR}/external/libs/libdatachannel") AND (${CMAKE_PROJECT_VERSION} STREQUAL "")))
        if(IS_TOP_LEVEL_VERSION_NEEDED)
            set(CMAKE_PROJECT_VERSION ${PROJECT_VERSION})
        endif()
        add_subdirectory("${SUBPROJECT_PATH}")
        if(IS_TOP_LEVEL_VERSION_NEEDED)
            unset(CMAKE_PROJECT_VERSION)
        endif()
    endif()
endforeach()

file(GLOB_RECURSE WEBRTC_TO_RTP_SOURCES src/*.cpp)
file(GLOB_RECURSE WEBRTC_TO_RTP_HEADERS src/*.hpp)
file(GLOB_RECURSE WEBRTC_TO_RTP_TEMPLATES src/*.tpp)


add_executable(${PROJECT_NAME})

target_compile_definitions(${PROJECT_NAME} PRIVATE WEBRTC_POSIX)

file(GLOB LIB_SDP_TRANSFORM_SOURCES ${LIBSDPTRANSFORM_DIR}/src/*.cpp)
target_sources(${PROJECT_NAME} PRIVATE ${WEBRTC_TO_RTP_HEADERS} ${WEBRTC_TO_RTP_SOURCES} ${LIB_SDP_TRANSFORM_SOURCES})

target_link_libraries(${PROJECT_NAME} 
    PRIVATE 
    datachannel CLI11::CLI11 Poco::Net Poco::NetSSL nlohmann_json::nlohmann_json
    )



target_include_directories(${PROJECT_NAME} PRIVATE ${LIBSDPTRANSFORM_DIR}/include)