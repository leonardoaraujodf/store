cmake_minimum_required(VERSION 3.10)
project(OnlineStore)

set(CMAKE_CXX_STANDARD 17)

add_executable(OnlineStore src/main.cpp)
target_include_directories(OnlineStore PRIVATE /usr/include/mysql)
target_link_libraries(OnlineStore mysqlcppconn)
