#include "UDPSender.hpp"

namespace Airlink {

UDPSender::UDPSender(std::string_view address, size_t port)
	: ISender(), serverAddress(std::make_unique<Poco::Net::SocketAddress>(address.data(), port)), socket(std::make_unique<Poco::Net::DatagramSocket>()),
	  address(address), port(port) {}

void UDPSender::sendData(std::vector<uint8_t> &data) { socket->sendTo(data.data(), data.size(), *serverAddress); }

void UDPSender::sendData(std::vector<uint8_t> &&data) { socket->sendTo(data.data(), data.size(), *serverAddress); }

void UDPSender::sendData(std::vector<uint8_t> &data) const { socket->sendTo(data.data(), data.size(), *serverAddress); }

void UDPSender::sendData(std::vector<uint8_t> &&data) const { socket->sendTo(data.data(), data.size(), *serverAddress); }

} // namespace Airlink