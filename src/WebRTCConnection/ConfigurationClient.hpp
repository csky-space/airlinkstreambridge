#ifndef CONFIGURATION_CLIENT_HPP
#define CONFIGURATION_CLIENT_HPP

#include <string>
#include <string_view>
#include <tuple>
#include <vector>

#include <QJsonDocument>

#include "../IceServerConfig.hpp"

class ConfigurationClient {
public:
    ConfigurationClient(std::string_view configurationServerUrl);
    std::tuple<std::vector<IceServerConfig>, std::vector<IceServerConfig>, std::string> getConfiguration() const;
private:
    void requestToLogin();
    void requestToConfig();

    void parseLogin();
    void parseConfig();

    QJsonDocument JSONReply;;

    std::string accessToken;
    std::string url;
    std::tuple<std::vector<IceServerConfig>, std::vector<IceServerConfig>, std::string> capturedConfigs;
};

#endif