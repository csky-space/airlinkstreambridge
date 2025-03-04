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


#include "rtc/h265nalunit.hpp"
#include "rtc/nalunit.hpp"
#include "rtc/rtcpnackresponder.hpp"
#include "rtc/rtcpsrreporter.hpp"
#include "rtc/rtpdepacketizer.hpp"
#include "rtc/rtppacketizationconfig.hpp"
#include "rtc/track.hpp"
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
#include <vector>


#include "../Transfer/ISender.hpp"

#define VIRTUAL_ID "4221"

WebRTCReceiver::WebRTCReceiver(std::shared_ptr<ISender> sender) 
    : IReceiver()
    , sender(sender)
    , parser()
    , config(std::make_unique<rtc::Configuration>())
    , ws(std::make_shared<rtc::WebSocket>())
{
    config->iceServers.emplace_back("stun:turn.air-link.space");
    rtc::IceServer turnzur("turn:turnzur.air-link.space");
    rtc::IceServer turn("turn:turn.air-link.space");
    turnzur.username = "airlink";
    turnzur.password = "lbjT3jXHt";
    turn.username = "airlink";
    turn.password = "lbjT3jXHt";

    config->iceServers.push_back(turn);
    config->iceServers.push_back(turnzur);

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
                        // Gathering complete, send answer
                        std::cout << "localDescription is: " << message.dump(0) << '\n';
                        ws->send(message.dump());
                    }
                }
            });

            peerConnection->onTrack([this](std::shared_ptr<rtc::Track> track){
                if(track->description().type() == "video") {
                    this->track = track;
                    std::cout << "adding a track\n";
                    //std::shared_ptr<rtc::RtpPacketizationConfig> rtpPacketizationConf = 
                    //    std::make_shared<rtc::RtpPacketizationConfig>(42, "video-send", 96, 90000);
                    //std::shared_ptr<rtc::RtpDe> session = 
                    //    std::make_shared<rtc::H265RtpPacketizer>(rtc::NalUnit::Separator::Length, rtpPacketizationConf);

                    //auto nackResponder = std::make_shared<rtc::RtcpNackResponder>();
                    //session->addToChain(nackResponder);
                    //track->setMediaHandler(session);

                    track->onMessage([this](std::variant<rtc::binary, std::string> data){
                        std::cout << "onMessage\n";
                        
                        rtc::binary bytedData = std::get<rtc::binary>(data);
                        std::cout << "message is: " << bytedData.size() << '\n';
                        std::vector<uint8_t> vData;
                        vData.resize(bytedData.size());
                        std::memcpy(vData.data(), bytedData.data(),  bytedData.size());
                        sender->sendData(std::move(vData));
                    });
                    
                    track->onFrame([this](const rtc::binary& frame, rtc::FrameInfo info){
                        //std::cout << "Received a frame\n";
                        //std::vector<uint8_t> vData;
                        //vData.resize(frame.size());
                        //std::memcpy(vData.data(), frame.data(),  frame.size());
                        //sender->sendData(std::move(vData));
                    });
                }
                
            });

            rtc::Description description(message["sdp"].get_string().value().data(), type);
            peerConnection->setRemoteDescription(description);

            peerConnection->setLocalDescription();
        }
	}
}