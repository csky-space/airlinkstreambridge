#ifndef WEBRTC_RECEIVER_HPP
#define WEBRTC_RECEIVER_HPP

#include <memory>
#include <string_view>

#include <simdjson.h>

#include "IReceiver.hpp"
#include "rtc/description.hpp"

namespace rtc {
    class WebSocket;
    class Configuration;
    class PeerConnection;
    class DataChannel;
    class Track;
    class H265RtpPacketizer;
};

class ISender;

class WebRTCReceiver : public IReceiver {
public:
    WebRTCReceiver(std::shared_ptr<ISender> sender);
    ~WebRTCReceiver();

    bool isOpened() override;
    bool isOpened() const override;
    bool isClosed() override;
    bool isClosed() const override;
protected:
    void onWsMessage(simdjson::ondemand::document& message);
private:
    constexpr static std::string_view wsUrl = "ws://signallingzur.air-link.space/wstest?name=GS001D0&partnerName=001D0";
    
    simdjson::ondemand::parser parser;
    std::unique_ptr<rtc::Configuration> config;
    std::unique_ptr<rtc::PeerConnection> peerConnection;
    std::shared_ptr<rtc::Track> track;
    std::shared_ptr<rtc::WebSocket> ws;
    std::shared_ptr<rtc::DataChannel> dataChannel;

    std::shared_ptr<ISender> sender;
};

#endif