#include "ConfigurationClient.hpp"

#include <QNetworkAccessManager>
#include <QJsonObject>
#include <QJsonDocument>
#include <qeventloop.h>
#include <qhttpheaders.h>
#include <qjsonobject.h>
#include <qnetworkaccessmanager.h>
#include <qobject.h>
#include <qvariant.h>
#include <QEventLoop>
#include <QNetworkReply>
#include <QJsonArray>

ConfigurationClient::ConfigurationClient(std::string_view configurationServerUrl) 
    : url(configurationServerUrl)
{
    requestToLogin();
    parseLogin();
    requestToConfig();
    parseConfig();
}

std::tuple<std::vector<IceServerConfig>, std::vector<IceServerConfig>, std::string> ConfigurationClient::getConfiguration() const {
    return capturedConfigs;
}

void ConfigurationClient::requestToLogin() {
    QNetworkAccessManager manager;
    QJsonObject jsonRequestObject;
    jsonRequestObject["name"] = "name";
    jsonRequestObject["password"] = "password";
    QJsonDocument document(jsonRequestObject);

    QNetworkRequest request(QString(url.data()) + "/GroundStation/login");
    request.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");

    QEventLoop loop;
    QNetworkReply* reply = manager.post(request, document.toJson());
    QObject::connect(reply, &QNetworkReply::finished, &loop, &QEventLoop::quit);
    
    loop.exec();
    JSONReply = QJsonDocument::fromJson(reply->readAll());
    reply->deleteLater();
}

void ConfigurationClient::requestToConfig() {
    QNetworkAccessManager manager;
    QJsonObject jsonRequestObject;
    jsonRequestObject["name"] = "name";
    jsonRequestObject["password"] = "password";
    QJsonDocument document(jsonRequestObject);

    QNetworkRequest request(QString(url.data()) + "/GroundStation/login");
    QHttpHeaders headers;
    headers.append(QHttpHeaders::WellKnownHeader::ContentType, "application/json");
    headers.append(QHttpHeaders::WellKnownHeader::Authorization, QString("Bearer ") + accessToken.c_str());
    request.setHeaders(headers);

    QEventLoop loop;
    QNetworkReply* reply = manager.post(request, document.toJson());
    QObject::connect(reply, &QNetworkReply::finished, &loop, &QEventLoop::quit);
    
    loop.exec();
    JSONReply = QJsonDocument::fromJson(reply->readAll());
    reply->deleteLater();
}

void ConfigurationClient::parseLogin() {
    accessToken = JSONReply["accessToken"].toString().toStdString();
}

void ConfigurationClient::parseConfig() {
    QJsonArray stunServers = JSONReply["stunServers"].toArray();
    for(const auto& server : stunServers) {
        std::get<0>(capturedConfigs).push_back({server.toString().toStdString()});
    }
    
    QJsonArray turnServers = JSONReply["turnServers"].toArray();
    for(const auto& server : turnServers) {
        QJsonObject serverObj = server.toObject();
        std::get<1>(capturedConfigs).push_back(
            {serverObj["url"].toString().toStdString(), 
                serverObj["username"].toString().toStdString(), 
                serverObj["credential"].toString().toStdString()});
    }
    
    std::get<2>(capturedConfigs) = JSONReply["wsUrl"].toString().toStdString();
}