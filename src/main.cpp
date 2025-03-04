#include <cstdint>
#include <cstring>
#include <ratio>
#include <vector>
#include <thread>
#include <chrono>
#include <memory>
#include <iostream>

#include "WebRTCConnection/Receive/WebRTCReceiver.hpp"
#include "WebRTCConnection/Transfer/RTPSender.hpp"
#include "WebRTCConnection/Transfer/UDPSender.hpp"

using std::chrono_literals::operator""ms;

int main(int argc, char** argv) {   
    std::shared_ptr<ISender> sender = std::make_shared<UDPSender>();
    //------------------------------------------------------------------------------
    std::unique_ptr<IReceiver> receiver = std::make_unique<WebRTCReceiver>(sender);

    //------------------------------------------------------------------------------

    while (!receiver->isOpened()) {
		if (receiver->isClosed())
			return 1;
		std::this_thread::sleep_for(100ms);
	}
    while(receiver->isOpened()) {

    }
    
    return 0;
}