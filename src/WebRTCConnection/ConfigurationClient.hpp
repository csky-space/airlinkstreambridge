#ifndef CONFIGURATION_CLIENT_HPP
#define CONFIGURATION_CLIENT_HPP

#include <string>
#include <string_view>
#include <memory>

//#include <Poco/Net/HTTPClientSession.h>
//#include <Poco/URI.h>
//
//#include <cpprest/http_client.h>

#include "WebrtcConfig.hpp"

namespace Airlink {

class ConfigurationClient {
public:
    ConfigurationClient(std::string_view configurationServerUrl, std::string_view login, std::string_view password);
    WebrtcConfiguration getConfiguration() const;
private:
    void requestToLogin();
    void requestToConfig();

    void parseLogin();

    //Poco::Net::HTTPClientSession session;
    //Poco::URI apiURI;

    std::string accessToken;
    
    WebrtcConfiguration configuration;

    std::string hostUrl;
    std::string login;
    std::string password;
    std::string prefix = "/groundStation";
};

}

#endif