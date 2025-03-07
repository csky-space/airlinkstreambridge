#include "ConfigurationClient.hpp"
#include "cpprest/http_msg.h"
#include "json.hpp"
#include <ostream>

ConfigurationClient::ConfigurationClient(std::string_view configurationServerUrl, std::string_view login, std::string_view password) 
    : client(std::make_unique<web::http::client::http_client>(configurationServerUrl.data()))
    , url(configurationServerUrl)
    , login(login)
    , password(password)
{ 
    requestToLogin();
    requestToConfig();
}

std::tuple<std::vector<IceServerConfig>, std::vector<IceServerConfig>, std::string> ConfigurationClient::getConfiguration() const {
    return capturedConfigs;
}

void ConfigurationClient::requestToLogin() {
    nlohmann::json jsonRequestObject;
    jsonRequestObject["name"] = login;
    jsonRequestObject["pass"] = password;
    
    web::http::http_request request(web::http::methods::POST);
    request.headers().set_content_type("application/json");
    request.set_body(jsonRequestObject.dump());
    request.set_request_uri(prefix + "/login");
    
    auto result = client->request(request);
    result.wait();
    
    auto response = result.get();
    if (response.status_code() != web::http::status_codes::OK) {
        throw std::runtime_error("Login failed with status: " + std::to_string(response.status_code()));
    }

    nlohmann::json replyBody = nlohmann::json::parse(response.extract_string().get());
    if (replyBody.contains("accessToken")) {
        accessToken = replyBody["accessToken"].get<std::string>();
    } else {
        throw std::runtime_error("No access token in response");
    }
}

void ConfigurationClient::requestToConfig() {
    web::http::http_request request(web::http::methods::GET);
    request.headers().set_content_type("application/json");
    request.set_request_uri(prefix + "/config");
    request.headers().add("Authorization", "Bearer " + accessToken);

    std::cout << "response is: " << request.to_string() << '\n';
    auto result = client->request(request);
    result.wait();
    auto response = result.get();
    if (response.status_code() != web::http::status_codes::OK) {
        throw std::runtime_error("Login failed with status: " + std::to_string(response.status_code()));
    }

    nlohmann::json replyBody = nlohmann::json::parse(response.extract_string().get());
    std::cout << "reply on get config" << replyBody.dump() << '\n';
    for(const auto& server : replyBody["stunServers"]) {
        std::get<0>(capturedConfigs).push_back({server, "", ""});
    }
    
    for(const auto& server : replyBody["turnServers"]) {
        std::get<1>(capturedConfigs).push_back(
            {server["url"], 
                server["username"], 
                server["credential"]});
    }
    
    std::get<2>(capturedConfigs) = replyBody["wsUrl"];
}