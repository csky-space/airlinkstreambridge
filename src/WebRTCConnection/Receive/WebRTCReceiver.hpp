#ifndef WEBRTC_RECEIVER_HPP
#define WEBRTC_RECEIVER_HPP

#include <cstddef>
#include <cstdint>
#include <memory>
#include <string_view>
#include <functional>
#include <chrono>

#include <simdjson.h>

#include "IReceiver.hpp"
#include "../WebrtcConfig.hpp"

using std::chrono_literals::operator""ms;

namespace rtc {
    class WebSocket;
    class Configuration;
    class PeerConnection;
    class DataChannel;
    class Track;
    class H265RtpPacketizer;
};

class ISender;

namespace Airlink {

class WebRTCReceiver : public IReceiver {
public:
    WebRTCReceiver(const std::vector<IceServerConfig>& stunServerConfigs, const std::vector<IceServerConfig>& turnServerConfigs, std::string_view signalUrl);
    explicit WebRTCReceiver(WebrtcConfiguration&& configuration) noexcept;
    ~WebRTCReceiver();
    
    bool isOpened() override;
    bool isOpened() const override;
    bool isClosed() override;
    bool isClosed() const override;
    
    void waitForConnection() override;
    
    void onData(const std::function<void(std::vector<uint8_t>&&)>& onVideoMessageAction) override;
    void onUpdate() override;
protected:
    void onWsMessage(simdjson::ondemand::document& message);
private:
    void connectWebRtc();
    void connectToSignallingServer();
    void createPeerConnection();
    
    std::string wsUrl;
    std::shared_ptr<rtc::WebSocket> ws;
    WebrtcConfiguration webrtcConfig;
        
    simdjson::ondemand::parser parser;
    
    std::unique_ptr<rtc::Configuration> config;
    std::unique_ptr<rtc::PeerConnection> peerConnection;
    std::shared_ptr<rtc::Track> track;
        
    std::shared_ptr<rtc::DataChannel> dataChannel;
    
    std::function<void(std::vector<uint8_t>&&)> onVideoMessageAction;
    
    std::string lastSDP = "";
    
    std::chrono::duration<long double, std::milli> failedTimeout = 5000ms;
    bool failed = false;
};

} //Airlink



#endif