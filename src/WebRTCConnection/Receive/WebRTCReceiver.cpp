#include "WebRTCReceiver.hpp"

#include <chrono>
#include <cstdint>
#include <cstring>
#include <iostream>
#include <memory>
#include <ostream>
#include <thread>
#include <variant>
#include <vector>

#include <rtc/common.hpp>
#include <rtc/configuration.hpp>
#include <rtc/datachannel.hpp>
#include <rtc/description.hpp>
#include <rtc/h265rtppacketizer.hpp>
#include <rtc/mediahandler.hpp>
#include <rtc/peerconnection.hpp>
#include <rtc/rtcpreceivingsession.hpp>
#include <rtc/track.hpp>
#include <rtc/websocket.hpp>

#include <sdptransform.hpp>

using std::chrono_literals::operator""ms;

namespace Airlink {
WebRTCReceiver::WebRTCReceiver(const std::vector<IceServerConfig> &stunServerConfigs, const std::vector<IceServerConfig> &turnServerConfigs,
							   std::string_view signalUrl, std::string_view sessionId)
	: IReceiver(), wsUrl(signalUrl), webrtcConfig{stunServerConfigs, turnServerConfigs, signalUrl.data(), sessionId.data()},
	  config(std::make_unique<rtc::Configuration>()) {

	connectToSignallingServer();
}

WebRTCReceiver::WebRTCReceiver(WebrtcConfiguration &&configuration) noexcept
	: IReceiver(), wsUrl(configuration.wsUrl), ws(std::make_shared<rtc::WebSocket>()), webrtcConfig(configuration),
	  config(std::make_unique<rtc::Configuration>()) {
	connectToSignallingServer();
}

WebRTCReceiver::~WebRTCReceiver() {}

bool WebRTCReceiver::isOpened() { return ws->isOpen(); }

bool WebRTCReceiver::isOpened() const { return ws->isOpen(); }

bool WebRTCReceiver::isClosed() { return ws->isClosed(); }

bool WebRTCReceiver::isClosed() const { return ws->isClosed(); }

void WebRTCReceiver::waitForConnection() {
	while (!isOpened()) {
		if (isClosed())
			break;
		std::this_thread::sleep_for(100ms);
	}
}

void WebRTCReceiver::onData(const std::function<void(std::vector<uint8_t> &&)> &onVideoMessageAction) { this->onVideoMessageAction = onVideoMessageAction; }

void WebRTCReceiver::onUpdate() {
	if(trackDataTimeout.elapsed<std::chrono::milliseconds>() > 5000) {
		trackDataTimeout.stop();
		peerConnection->close();
		while(peerConnection->state() != rtc::PeerConnection::State::Closed) {}
		std::cout << "ping\n";
		ws->send(json{{"id", webrtcConfig.sessionId}, {"type", "ping"}}.dump());
	}
}

void WebRTCReceiver::onWsMessage(const nlohmann::json &message) {
	std::cout << "parsed on message\n";

	auto idResult = message.find("id");
	std::cout << "message is: " << message.dump() << '\n' << std::flush;
	if (idResult == message.cend()) {
		return;
	}

	std::string id(idResult.value());

	auto typeResult = message.find("type");
	if (typeResult == message.cend()) {
		return;
	}

	std::string type(typeResult.value());

	if (type == "ping") {
		std::cout << "send request\n";
		ws->send(json{{"id", webrtcConfig.sessionId}, {"type", "request"}}.dump());
	}

	if (type == "offer") {
		std::cout << "offer\n";

		lastSDP = message["sdp"];
		createPeerConnection();
	}
}

void WebRTCReceiver::connectWebRtc() { connectToSignallingServer(); }

void WebRTCReceiver::connectToSignallingServer() {
	rtc::WebSocketConfiguration wsConfig;
	wsConfig.connectionTimeout = std::chrono::duration<uint32_t>(30);
	ws = std::make_shared<rtc::WebSocket>(wsConfig);
	for (const auto &serverConfig : webrtcConfig.stunServerConfigs) {
		config->iceServers.emplace_back(serverConfig.url);
	}
	for (const auto &serverConfig : webrtcConfig.turnServersConfigs) {
		rtc::IceServer server(serverConfig.url);
		server.username = serverConfig.login;
		server.password = serverConfig.password;
		config->iceServers.push_back(server);
	}

	config->disableAutoNegotiation = true;
	ws->onOpen([this]() {
		std::cout << "ping\n";
		ws->send(json{{"id", webrtcConfig.sessionId}, {"type", "ping"}}.dump());
	});

	ws->onClosed([]() { std::cout << "WebSocket closed" << std::endl; });

	ws->onError([](const std::string &error) { std::cout << "WebSocket failed: " << error << std::endl; });

	ws->onMessage([&](std::variant<rtc::binary, std::string> data) {
		if (!std::get_if<std::string>(&data)) {
			std::cout << "unsupported message\n";
			return;
		}
		nlohmann::json message = nlohmann::json::parse(std::get<std::string>(data));
		onWsMessage(message);
	});

	ws->open(wsUrl);
}

void WebRTCReceiver::createPeerConnection() {
	peerConnection = std::make_unique<rtc::PeerConnection>(*config);

	peerConnection->onStateChange([this](rtc::PeerConnection::State state) {
		std::cout << "State: " << state << std::endl;
		if (state == rtc::PeerConnection::State::Disconnected || state == rtc::PeerConnection::State::Failed || state == rtc::PeerConnection::State::Closed) {
			if (failed) {
				std::this_thread::sleep_for(15000ms);
				std::cout << "ping\n";
				ws->send(json{{"id", webrtcConfig.sessionId}, {"type", "ping"}}.dump());
				failed = false;
			}
		}
		if (state == rtc::PeerConnection::State::Failed) {
			failed = true;
		}
	});
	peerConnection->onGatheringStateChange([this](rtc::PeerConnection::GatheringState state) {
		if (state == rtc::PeerConnection::GatheringState::Complete) {
			if (peerConnection) {
				std::cout << "gathering complete\n";

				auto description = peerConnection->localDescription();
				std::cout << "description type is " << description->typeString() << '\n';
				json message = {{"id", webrtcConfig.sessionId}, {"type", description->typeString()}, {"sdp", description.value()}};

				std::cout << "localDescription is: " << message.dump(0) << '\n';
				ws->send(message.dump());
			}
		}
	});

	peerConnection->onTrack([this](std::shared_ptr<rtc::Track> track) {
		if (track->description().type() == "video") {
			this->track = track;
			std::cout << "adding a track\n";

			track->onMessage([this](std::variant<rtc::binary, std::string> data) {
				trackDataTimeout.restart();
				rtc::binary bytedData = std::get<rtc::binary>(data);
				std::vector<uint8_t> vData;
				vData.resize(bytedData.size());
				std::memcpy(vData.data(), bytedData.data(), bytedData.size());
				if (onVideoMessageAction)
					onVideoMessageAction(std::move(vData));
			});
		}
	});
	// peerConnection->onDataChannel([this](std::shared_ptr<rtc::DataChannel> dataChannel){
	//	this->dataChannel = dataChannel;
	//	dataChannel->onOpen([dataChannel]() {
	//
	//	});
	//
	//	dataChannel->onMessage(nullptr, [dataChannel](std::string msg) {
	//		std::cout << "received message from datachannel: " << msg << '\n';
	//	});
	// });

	rtc::Description description(lastSDP, "offer");
	peerConnection->setRemoteDescription(description);

	peerConnection->setLocalDescription();
}

} // namespace Airlink