#include <bits/chrono.h>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <memory>
#include <thread>
#include <utility>
#include <vector>

#include <CLI/CLI.hpp>

#include "WebRTCConnection/ConfigurationClient.hpp"
#include "WebRTCConnection/Receive/IReceiver.hpp"
#include "WebRTCConnection/Receive/WebRTCReceiver.hpp"
#include "WebRTCConnection/Transfer/UDPSender.hpp"

using std::chrono_literals::operator""ms;

int main(int argc, char **argv) {
	//==============================================================================
	CLI::App app{"AirlinkStreamBridge"};
	argv = app.ensure_utf8(argv);

	std::string apiURL;
	std::string modemName;
	std::string password;
	std::vector<std::string> modemNames;
	size_t udpPort = 5000;

	app.add_option("-a,--api-url", apiURL, "provide the api server url for a getting configuration");
	app.add_option("-m,--modem-name", modemName, "provide the Air-link modem name");
	app.add_option("-p,--password", password, "provide a password for the authorization");
	app.add_option("-s,--stream-port", udpPort, "The bridge will streams to this port. It doesn't work yet");

	// for multi modems support
	// app.add_option("-mn", modemNames, "modem-names");

	CLI11_PARSE(app, argc, argv);
	//==============================================================================
	std::shared_ptr<Airlink::ISender> sender = std::make_shared<Airlink::UDPSender>("127.0.0.1", udpPort);
	//------------------------------------------------------------------------------
	Airlink::ConfigurationClient client(apiURL, modemName, password);
	auto webrtcConfig = client.getConfiguration();
	webrtcConfig.wsUrl += std::string("?name=GS") + modemName + "&partnerName=" + modemName;
	std::shared_ptr<Airlink::IReceiver> receiver = std::make_shared<Airlink::WebRTCReceiver>(std::move(webrtcConfig));

	receiver->onData([sender](std::vector<uint8_t> &&videoMessage) { sender->sendData(videoMessage); });
	//------------------------------------------------------------------------------

	receiver->waitForConnection();

	while (receiver->isOpened()) {
		std::this_thread::sleep_for(100ms);
		// receiver->onUpdate();
	}

	return 0;
}