#include "WebRTCReceiver.hpp"

#include <cstdint>
#include <cstring>
#include <iostream>
#include <memory>
#include <ostream>
#include <variant>
#include <vector>
#include <thread>
#include <chrono>

#include <rtc/track.hpp>
#include <rtc/datachannel.hpp>
#include <rtc/description.hpp>
#include <rtc/configuration.hpp>
#include <rtc/websocket.hpp>
#include <rtc/peerconnection.hpp>
#include <rtc/mediahandler.hpp>
#include <rtc/rtcpreceivingsession.hpp>
#include <rtc/common.hpp>
#include <rtc/h265rtppacketizer.hpp>

#include <sdptransform.hpp>

#include <simdjson.h>

#define VIRTUAL_ID "4221"

using std::chrono_literals::operator""ms;

WebRTCReceiver::WebRTCReceiver(const std::vector<IceServerConfig>& stunServerConfigs, const std::vector<IceServerConfig>& turnServerConfigs, std::string_view signalUrl)
    : IReceiver()
    , wsUrl(signalUrl)
    , parser()
    , config(std::make_unique<rtc::Configuration>())
    , ws(std::make_shared<rtc::WebSocket>())
{
    for(const auto& serverConfig : stunServerConfigs) {
        config->iceServers.emplace_back(serverConfig.url);
    }
    for(const auto& serverConfig : turnServerConfigs) {
        rtc::IceServer server(serverConfig.url);
        server.username = serverConfig.login;
        server.password = serverConfig.password;
        config->iceServers.push_back(server);
    }

    config->disableAutoNegotiation = true;

    ws->onOpen([this](){
        std::cout << "ping";
        ws->send(json{{"id", VIRTUAL_ID}, {"type", "ping"}}.dump());
    });

    ws->onClosed([]() { std::cout << "WebSocket closed" << std::endl; });

	ws->onError([](const std::string &error) { std::cout << "WebSocket failed: " << error << std::endl; });

	ws->onMessage([&](std::variant<rtc::binary, std::string> data) {
		if (!std::get_if<std::string>(&data)) {
            std::cout << "unsupported message\n";
            return;
        }
        simdjson::ondemand::document message = parser.iterate(std::get<std::string>(data));
        onWsMessage(message);
	});

    ws->open(wsUrl);
}

WebRTCReceiver::WebRTCReceiver(WebrtcConfiguration&& configuration) noexcept 
    : IReceiver()
    , wsUrl(configuration.wsUrl)
    , parser()
    , config(std::make_unique<rtc::Configuration>())
    , ws(std::make_shared<rtc::WebSocket>())
{
    for(const auto& serverConfig : configuration.stunServerConfigs) {
        config->iceServers.emplace_back(serverConfig.url);
    }
    for(const auto& serverConfig : configuration.turnServersConfigs) {
        rtc::IceServer server(serverConfig.url);
        server.username = serverConfig.login;
        server.password = serverConfig.password;
        config->iceServers.push_back(server);
    }

    config->disableAutoNegotiation = true;

    ws->onOpen([this](){
        std::cout << "ping";
        ws->send(json{{"id", VIRTUAL_ID}, {"type", "ping"}}.dump());
    });

    ws->onClosed([]() { std::cout << "WebSocket closed" << std::endl; });

	ws->onError([](const std::string &error) { std::cout << "WebSocket failed: " << error << std::endl; });

	ws->onMessage([&](std::variant<rtc::binary, std::string> data) {
		if (!std::get_if<std::string>(&data)) {
            std::cout << "unsupported message\n";
            return;
        }
        simdjson::ondemand::document message = parser.iterate(std::get<std::string>(data));
        onWsMessage(message);
	});

    ws->open(wsUrl);
}

WebRTCReceiver::~WebRTCReceiver() {
    
}

bool WebRTCReceiver::isOpened() {
    return ws->isOpen();
}

bool WebRTCReceiver::isOpened() const {
    return ws->isOpen();
}

bool WebRTCReceiver::isClosed() {
    return ws->isClosed();
}

bool WebRTCReceiver::isClosed() const {
    return ws->isClosed();
}

void WebRTCReceiver::waitForConnection() {
    while (!isOpened()) {
		if (isClosed())
			break;
		std::this_thread::sleep_for(100ms);
	}
}

void WebRTCReceiver::onData(const std::function<void(std::vector<uint8_t>&&)>& onVideoMessageAction) {
    this->onVideoMessageAction = onVideoMessageAction;
}

void WebRTCReceiver::onWsMessage(simdjson::ondemand::document& message) {
    std::cout << "parsed on message\n";

    auto idResult = message.find_field("id");
    if(idResult.error() != simdjson::SUCCESS) {
        return;
    }

    std::string id(idResult.get_string().value());


    auto typeResult = message.find_field("type");
    if(typeResult.error() != simdjson::SUCCESS) {
        return;
    }

    std::string type(typeResult.get_string().value());

    if (type == "ping") {
        std::cout << "send request\n";
        ws->send(json{{"id", VIRTUAL_ID}, {"type", "request"}}.dump());
	}

	if (type == "offer") {
		std::cout << "offer\n";

        if(!peerConnection) {
            peerConnection = std::make_unique<rtc::PeerConnection>(*config);
            
            peerConnection->onStateChange([](rtc::PeerConnection::State state){
                if (state == rtc::PeerConnection::State::Disconnected || state == rtc::PeerConnection::State::Failed ||
                    state == rtc::PeerConnection::State::Closed) {
                        std::cout << "State: " << state << std::endl;
                    }
            });
            peerConnection->onGatheringStateChange([this](rtc::PeerConnection::GatheringState state){
                if (state == rtc::PeerConnection::GatheringState::Complete) {
                    if (peerConnection) {
                        std::cout << "gathering complete\n";
                        
                        auto description = peerConnection->localDescription();
                        std::cout << "description type is " << description->typeString() << '\n';
                        json message = {{"id", VIRTUAL_ID}, {"type", description->typeString()}, {"sdp", description.value()}};

                        std::cout << "localDescription is: " << message.dump(0) << '\n';
                        ws->send(message.dump());
                    }
                }
            });

            peerConnection->onTrack([this](std::shared_ptr<rtc::Track> track){
                if(track->description().type() == "video") {
                    this->track = track;
                    std::cout << "adding a track\n";

                    track->onMessage([this](std::variant<rtc::binary, std::string> data){                        
                        rtc::binary bytedData = std::get<rtc::binary>(data);
                        std::vector<uint8_t> vData;
                        vData.resize(bytedData.size());
                        std::memcpy(vData.data(), bytedData.data(),  bytedData.size());
                        if(onVideoMessageAction)
                            onVideoMessageAction(std::move(vData));
                    });
                }
                
            });

            rtc::Description description(message["sdp"].get_string().value().data(), type);
            peerConnection->setRemoteDescription(description);

            peerConnection->setLocalDescription();
        }
	}
}