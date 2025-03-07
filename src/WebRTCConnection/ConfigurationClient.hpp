#ifndef CONFIGURATION_CLIENT_HPP
#define CONFIGURATION_CLIENT_HPP

#include <string>
#include <string_view>
#include <tuple>
#include <vector>
#include <memory>

#include <cpprest/http_client.h>

#include "../IceServerConfig.hpp"



class ConfigurationClient {
public:
    ConfigurationClient(std::string_view configurationServerUrl, std::string_view login, std::string_view password);
    std::tuple<std::vector<IceServerConfig>, std::vector<IceServerConfig>, std::string> getConfiguration() const;
private:
    void requestToLogin();
    void requestToConfig();

    void parseLogin();

    std::unique_ptr<web::http::client::http_client> client;

    std::string accessToken;
    
    std::tuple<std::vector<IceServerConfig>, std::vector<IceServerConfig>, std::string> capturedConfigs;

    std::string url;
    std::string login;
    std::string password;
    std::string prefix = "/groundStation";
};

#endif