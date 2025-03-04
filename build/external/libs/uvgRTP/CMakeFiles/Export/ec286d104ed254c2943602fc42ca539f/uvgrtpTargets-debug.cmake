#----------------------------------------------------------------
# Generated CMake target import file for configuration "Debug".
#----------------------------------------------------------------

# Commands may need to know the format version.
set(CMAKE_IMPORT_FILE_VERSION 1)

# Import target "uvgrtp::uvgrtp" for configuration "Debug"
set_property(TARGET uvgrtp::uvgrtp APPEND PROPERTY IMPORTED_CONFIGURATIONS DEBUG)
set_target_properties(uvgrtp::uvgrtp PROPERTIES
  IMPORTED_LOCATION_DEBUG "${_IMPORT_PREFIX}/lib/libuvgrtp.so.3.1.5;+;-source"
  IMPORTED_SONAME_DEBUG "libuvgrtp.so.3"
  )

list(APPEND _cmake_import_check_targets uvgrtp::uvgrtp )
list(APPEND _cmake_import_check_files_for_uvgrtp::uvgrtp "${_IMPORT_PREFIX}/lib/libuvgrtp.so.3.1.5;+;-source" )

# Commands beyond this point should not need to know the version.
set(CMAKE_IMPORT_FILE_VERSION)
