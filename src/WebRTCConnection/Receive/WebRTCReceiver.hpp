#ifndef WEBRTC_RECEIVER_HPP
#define WEBRTC_RECEIVER_HPP

#include <cstdint>
#include <memory>
#include <string_view>
#include <functional>

#include <simdjson.h>

#include "IReceiver.hpp"
#include "../../IceServerConfig.hpp"


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
    WebRTCReceiver(std::span<IceServerConfig> stunUrls, std::span<IceServerConfig> turnUrls, std::string_view signalUrl);
    ~WebRTCReceiver();

    bool isOpened() override;
    bool isOpened() const override;
    bool isClosed() override;
    bool isClosed() const override;

    void onVideoMessage(const std::function<void(std::vector<uint8_t>&&)>& onVideoMessageAction);
protected:
    void onWsMessage(simdjson::ondemand::document& message);
private:
    std::string wsUrl;
    
    simdjson::ondemand::parser parser;

    std::unique_ptr<rtc::Configuration> config;
    std::unique_ptr<rtc::PeerConnection> peerConnection;
    std::shared_ptr<rtc::Track> track;
    std::shared_ptr<rtc::WebSocket> ws;
    std::shared_ptr<rtc::DataChannel> dataChannel;

    std::function<void(std::vector<uint8_t>&&)> onVideoMessageAction;
};

#endif