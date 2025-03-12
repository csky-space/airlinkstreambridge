#include "ConfigurationClient.hpp"

#include <ostream>
#include <sstream>

#include "json.hpp"

#include <Poco/Net/SSLManager.h>
#include <Poco/Net/HTTPRequest.h>
#include <Poco/Net/HTTPResponse.h>
#include <Poco/Net/AcceptCertificateHandler.h>
#include <Poco/StreamCopier.h>




namespace Airlink {

ConfigurationClient::ConfigurationClient(std::string_view configurationServerUrl, std::string_view login, std::string_view password) 
    : session(configurationServerUrl.data(), 443)
    , apiURI(std::string(configurationServerUrl))
    , hostUrl(apiURI.getHost())
    , login(login)
    , password(password)
{ 
    Poco::Net::initializeSSL();
    Poco::SharedPtr<Poco::Net::InvalidCertificateHandler> pCertHandler = 
            new Poco::Net::AcceptCertificateHandler(true);
    pContext = 
            new Poco::Net::Context(Poco::Net::Context::CLIENT_USE, "", "", "", 
                                  Poco::Net::Context::VERIFY_NONE, 9, false, 
                                  "ALL:!ADH:!LOW:!EXP:!MD5:@STRENGTH");
        Poco::Net::SSLManager::instance().initializeClient(0, pCertHandler, pContext);
    session.setKeepAlive(true);
    requestToLogin();
    requestToConfig();
}

WebrtcConfiguration ConfigurationClient::getConfiguration() const {
    return configuration;
}

void ConfigurationClient::requestToLogin() {
    nlohmann::json jsonRequestObject;
    jsonRequestObject["name"] = login;
    jsonRequestObject["pass"] = password;
    const std::string body = jsonRequestObject.dump();

    Poco::Net::HTTPRequest request(Poco::Net::HTTPRequest::HTTP_POST, queryPrefix + "/login", Poco::Net::HTTPMessage::HTTP_1_1);
    request.setContentType("application/json");
    
    request.setContentLength(body.length());
    std::cout << "host is: " << session.getHost() << '\n';
    
    std::ostream& os = session.sendRequest(request);
    os  << body;

    Poco::Net::HTTPResponse response;
    std::istream& responseStream = session.receiveResponse(response);
    std::ostringstream responseData;
    Poco::StreamCopier::copyStream(responseStream, responseData);

    std::cout << "response: " + responseData.str() + '\n';
    nlohmann::json replyBody = nlohmann::json::parse(responseData.str());
    if (replyBody.contains("accessToken")) {
        accessToken = replyBody["accessToken"].get<std::string>();
    } else {
        throw std::runtime_error("No access token in response");
    }
}

void ConfigurationClient::requestToConfig() {    
    Poco::Net::HTTPRequest request(Poco::Net::HTTPRequest::HTTP_GET, queryPrefix + "/config", Poco::Net::HTTPMessage::HTTP_1_1);
    request.add("Authorization", "Bearer " + accessToken);
    request.setContentType("application/json");

    std::cout << "host is: " << session.getHost() << '\n';

    session.sendRequest(request);

    Poco::Net::HTTPResponse response;
    std::istream& responseStream = session.receiveResponse(response);
    std::ostringstream responseData;
    Poco::StreamCopier::copyStream(responseStream, responseData);

    std::cout << "response: " + responseData.str() + '\n';
    nlohmann::json replyBody = nlohmann::json::parse(responseData.str());

    std::cout << "reply on get config" << replyBody.dump() << '\n';
    for(const auto& server : replyBody["stunServers"]) {
        configuration.stunServerConfigs.push_back({server, "", ""});
    }
    
    for(const auto& server : replyBody["turnServers"]) {
        configuration.turnServersConfigs.push_back(
            {server["url"], 
                server["username"], 
                server["credential"]});
    }
    
    configuration.wsUrl = replyBody["wsUrl"];
}

}