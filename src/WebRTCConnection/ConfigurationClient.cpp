#include "ConfigurationClient.hpp"

#include <ostream>
#include <sstream>

#include "json.hpp"

//#include <Poco/Net/HTTPRequest.h>
//#include <Poco/Net/HTTPResponse.h>
//#include <Poco/StreamCopier.h>
//#include <Poco/Net/Net.h>
//
//#include "cpprest/http_msg.h"




namespace Airlink {

ConfigurationClient::ConfigurationClient(std::string_view configurationServerUrl, std::string_view login, std::string_view password) 
    : //session(configurationServerUrl.data(), 443)
    //, //apiURI(configurationServerUrl.data())
     hostUrl(configurationServerUrl)
    , login(login)
    , password(password)
{ 
    //Poco::Net::initializeSSL();
    //session.setKeepAlive(true);
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

    //Poco::Net::HTTPRequest request(Poco::Net::HTTPRequest::HTTP_POST, apiURI.getPathAndQuery() + prefix + "/login", Poco::Net::HTTPMessage::HTTP_1_1);
    //request.setContentType("application/json");
    //
    //request.setContentLength(body.length());
    //std::cout << "host is: " << session.getHost() << '\n';
    //
    //std::ostream& os = session.sendRequest(request);
    //os  << body;
//
    //Poco::Net::HTTPResponse response;
    //std::istream& responseStream = session.receiveResponse(response);
    //std::ostringstream responseData;
    //Poco::StreamCopier::copyStream(responseStream, responseData);
    
    //web::http::http_request request(web::http::methods::POST);
    //request.headers().set_content_type("application/json");
    //request.set_body(jsonRequestObject.dump());
    //request.set_request_uri(prefix + "/login");
    //
    //auto result = client->request(request);
    //result.wait();
    //
    //auto response = result.get();
    //if (response.status_code() != web::http::status_codes::OK) {
    //    throw std::runtime_error("Login failed with status: " + std::to_string(response.status_code()));
    //}

    //std::cout << "response: " + responseData.str() + '\n';
    //nlohmann::json replyBody = nlohmann::json::parse(responseData.str());
    //if (replyBody.contains("accessToken")) {
    //    accessToken = replyBody["accessToken"].get<std::string>();
    //} else {
    //    throw std::runtime_error("No access token in response");
    //}
}

void ConfigurationClient::requestToConfig() {    
    //web::http::http_request request(web::http::methods::GET);
    //request.headers().set_content_type("application/json");
    //request.set_request_uri(prefix + "/config");
    //request.headers().add("Authorization", "Bearer " + accessToken);
//
    //std::cout << "response is: " << request.to_string() << '\n';
    //auto result = client->request(request);
    //result.wait();
    //auto response = result.get();
    //if (response.status_code() != web::http::status_codes::OK) {
    //    throw std::runtime_error("Login failed with status: " + std::to_string(response.status_code()));
    //}
//
    //nlohmann::json replyBody = nlohmann::json::parse(response.extract_string().get());
    //std::cout << "reply on get config" << replyBody.dump() << '\n';
    //for(const auto& server : replyBody["stunServers"]) {
    //    configuration.stunServerConfigs.push_back({server, "", ""});
    //}
    //
    //for(const auto& server : replyBody["turnServers"]) {
    //    configuration.turnServersConfigs.push_back(
    //        {server["url"], 
    //            server["username"], 
    //            server["credential"]});
    //}
    //
    //configuration.wsUrl = replyBody["wsUrl"];
}

}