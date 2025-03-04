# Additional clean files
cmake_minimum_required(VERSION 3.16)

if("${CONFIG}" STREQUAL "" OR "${CONFIG}" STREQUAL "Debug")
  file(REMOVE_RECURSE
  "CMakeFiles/WebRTCtoRtp_autogen.dir/AutogenUsed.txt"
  "CMakeFiles/WebRTCtoRtp_autogen.dir/ParseCache.txt"
  "WebRTCtoRtp_autogen"
  )
endif()
