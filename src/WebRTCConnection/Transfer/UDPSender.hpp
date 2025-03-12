#ifndef UDP_SENDER_HPP
#define UDP_SENDER_HPP

#include <cstddef>
#include <cstdint>
#include <memory>
#include <string>

#include "ISender.hpp"

#include <Poco/Net/DatagramSocket.h>
#include <Poco/Net/SocketAddress.h>

// namespace Poco::Net {
//     class SocketAddress;
//     class DatagramSocket;
// }

namespace Airlink {

class UDPSender : public ISender {
  public:
	UDPSender(std::string_view address = "127.0.0.1", size_t port = 5000);

	void sendData(std::vector<uint8_t> &data);
	void sendData(std::vector<uint8_t> &&data);
	void sendData(std::vector<uint8_t> &data) const;
	void sendData(std::vector<uint8_t> &&data) const;

  private:
	std::unique_ptr<Poco::Net::SocketAddress> serverAddress;
	std::unique_ptr<Poco::Net::DatagramSocket> socket;

	std::string address;
	size_t port;
};

} // namespace Airlink

#endif