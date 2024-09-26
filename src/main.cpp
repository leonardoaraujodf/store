#include <mysql_driver.h>
#include <mysql_connection.h>
#include <cppconn/statement.h>
#include <cppconn/resultset.h>
#include <iostream>
#include <thread>
#include <chrono>

int main() {
    sql::mysql::MySQL_Driver *driver;
    sql::Connection *con;
    int retries = 5;
    int delay = 5; // seconds

    driver = sql::mysql::get_mysql_driver_instance();

    for (int i = 0; i < retries; ++i) {
        try {
            con = driver->connect("tcp://db:3306", "root", "password");
            con->setSchema("storedb");
            std::cout << "Successfully connected to the database!" << std::endl;
            delete con;
            while (1) {
            }
            return 0;
        } catch (sql::SQLException &e) {
            std::cerr << "Error connecting to the database: " << e.what() << std::endl;
            std::cerr << "Retrying in " << delay << " seconds..." << std::endl;
            std::this_thread::sleep_for(std::chrono::seconds(delay));
        }
    }

    std::cerr << "Failed to connect to the database after " << retries << " attempts." << std::endl;
    return 1;
}
