#include "WebRTCReceiver.hpp"

#include <cstddef>
#include <cstdint>
#include <cstring>
#include <iostream>
#include <fstream>
#include <memory>
#include <ostream>
#include <sstream>
#include <variant>
#include <vector>

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

#include "../Transfer/ISender.hpp"

#define VIRTUAL_ID "4221"

WebRTCReceiver::WebRTCReceiver(std::span<IceServerConfig> stunUrls, std::span<IceServerConfig> turnUrls, std::string_view signalUrl)
    : IReceiver()
    , parser()
    , config(std::make_unique<rtc::Configuration>())
    , ws(std::make_shared<rtc::WebSocket>())
    , wsUrl(signalUrl)
{
    for(const auto& serverConfig : stunUrls) {
        config->iceServers.emplace_back(serverConfig.url);
    }
    for(const auto& serverConfig : turnUrls) {
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
			

		std::cout << "onMessage\n";
        simdjson::ondemand::document message = parser.iterate(std::get<std::string>(data));
        onWsMessage(message);
	});

    ws->open(wsUrl.data());
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

void WebRTCReceiver::onVideoMessage(const std::function<void(std::vector<uint8_t>&&)>& onVideoMessageAction) {
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
    
        std::fstream f("/home/szamaro/Projects/WebRTCtoLocalRTP/build/offer.sdp");

        if(f.is_open() && !peerConnection) {
            std::stringstream buffer;
            buffer << f.rdbuf();
            
            auto rtspSDPJSON = sdptransform::parse((buffer.str()));
            std::cout << rtspSDPJSON.dump(4) << std::endl;
            f.close();
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
                        std::cout << "onMessage\n";
                        
                        rtc::binary bytedData = std::get<rtc::binary>(data);
                        std::cout << "message is: " << bytedData.size() << '\n';
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