#include "UDPSender.hpp"

#include <iostream>

namespace Airlink {

UDPSender::UDPSender(std::string_view address, size_t port)
	: ISender(), serverAddress(std::make_unique<Poco::Net::SocketAddress>(address.data(), port)), socket(std::make_unique<Poco::Net::DatagramSocket>()),
	  address(address), port(port) {
		socket->connect(*serverAddress);
	  }

void UDPSender::sendData(std::vector<uint8_t> &data) { 
	socket->sendBytes(data.data(), data.size()); }

void UDPSender::sendData(std::vector<uint8_t> &&data) { socket->sendBytes(data.data(), data.size()); }

void UDPSender::sendData(std::vector<uint8_t> &data) const { socket->sendBytes(data.data(), data.size()); }

void UDPSender::sendData(std::vector<uint8_t> &&data) const { socket->sendBytes(data.data(), data.size()); }

} // namespace Airlink