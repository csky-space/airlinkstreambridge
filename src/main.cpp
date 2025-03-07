#include <cstdint>
#include <cstring>
#include <qcoreapplication.h>
#include <utility>
#include <vector>
#include <thread>
#include <chrono>
#include <memory>
#include <iostream>

#include <QCoreApplication>

#include "WebRTCConnection/ConfigurationClient.hpp"
#include "WebRTCConnection/Receive/IReceiver.hpp"
#include "WebRTCConnection/Receive/WebRTCReceiver.hpp"
#include "WebRTCConnection/Transfer/UDPSender.hpp"

using std::chrono_literals::operator""ms;

int main() {  
    std::shared_ptr<ISender> sender = std::make_shared<UDPSender>();
    //------------------------------------------------------------------------------
    ConfigurationClient client("https://stage.air-link.space/api/", "001D0", "HM9-qc5-Ddq-mgy");
    auto configs = client.getConfiguration();
    std::shared_ptr<WebRTCReceiver> webrtcReceiver = 
    std::make_shared<WebRTCReceiver>(std::get<0>(configs), std::get<1>(configs), std::get<2>(configs));
    
    webrtcReceiver->onVideoMessage([sender](std::vector<uint8_t>&& videoMessage){
        sender->sendData(std::move(videoMessage));
    });
    //------------------------------------------------------------------------------
    std::shared_ptr<IReceiver> receiver = webrtcReceiver;

    while (!webrtcReceiver->isOpened()) {
		if (webrtcReceiver->isClosed())
			return 1;
		std::this_thread::sleep_for(100ms);
	}
    while(webrtcReceiver->isOpened()) {

    }
    
    return 0;
}