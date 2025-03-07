#ifndef CONFIGURATION_CLIENT_HPP
#define CONFIGURATION_CLIENT_HPP

#include <string>
#include <string_view>
#include <memory>

#include <cpprest/http_client.h>

#include "../WebrtcConfig.hpp"

class ConfigurationClient {
public:
    ConfigurationClient(std::string_view configurationServerUrl, std::string_view login, std::string_view password);
    WebrtcConfiguration getConfiguration() const;
private:
    void requestToLogin();
    void requestToConfig();

    void parseLogin();

    std::unique_ptr<web::http::client::http_client> client;

    std::string accessToken;
    
    WebrtcConfiguration configuration;

    std::string url;
    std::string login;
    std::string password;
    std::string prefix = "/groundStation";
};

#endif