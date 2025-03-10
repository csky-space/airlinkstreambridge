#include <cstdint>
#include <cstring>
#include <qcoreapplication.h>
#include <utility>
#include <vector>
#include <memory>

#include <CLI/CLI.hpp>

#include "WebRTCConnection/ConfigurationClient.hpp"
#include "WebRTCConnection/Receive/IReceiver.hpp"
#include "WebRTCConnection/Receive/WebRTCReceiver.hpp"
#include "WebRTCConnection/Transfer/UDPSender.hpp"

int main(int argc, char** argv) {  
    //==============================================================================
    CLI::App app{"AirlinkStreamBridge"};
    argv = app.ensure_utf8(argv);
    
    std::string apiURL;
    std::string modemName;
    std::string password;
    std::vector<std::string> modemNames;

    app.add_option("-a,--api-url", apiURL, "provide the api server url for a getting configuration");
    app.add_option("-m,--modem-name", modemName, "provide the Air-link modem name");
    app.add_option("-p,--password", password, "provide a password for the authorization");
    //for multi modems support
    //app.add_option("-mn", modemNames, "modem-names");

    CLI11_PARSE(app, argc, argv);
    //==============================================================================
    std::shared_ptr<ISender> sender = std::make_shared<UDPSender>();
    //------------------------------------------------------------------------------
    ConfigurationClient client(apiURL, modemName, password);
    auto webrtcConfig = client.getConfiguration();
    webrtcConfig.wsUrl += std::string("?name=GS") + modemName + "&partnerName=" + modemName; 
    std::shared_ptr<IReceiver> receiver = std::make_shared<WebRTCReceiver>(std::move(webrtcConfig));
    
    receiver->onData([sender](std::vector<uint8_t>&& videoMessage){
        sender->sendData(videoMessage);
    });
    //------------------------------------------------------------------------------
    
    receiver->waitForConnection();
    
    while(true) {
        receiver->onUpdate();
    }
    
    return 0;
}