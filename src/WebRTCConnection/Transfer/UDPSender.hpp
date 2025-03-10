#ifndef UDP_SENDER_HPP
#define UDP_SENDER_HPP


#include <cstddef>
#include <cstdint>

#include <QUdpSocket>
#include <QHostAddress>

#include "ISender.hpp"

namespace Airlink {

class UDPSender : public ISender {
public:
    UDPSender(QHostAddress address = QHostAddress::SpecialAddress::LocalHost, size_t port = 5000);

    void sendData(std::vector<uint8_t>& data);
    void sendData(std::vector<uint8_t>&& data);
    void sendData(std::vector<uint8_t>& data) const;
    void sendData(std::vector<uint8_t>&& data) const;
private:
    QUdpSocket socket;
    QHostAddress address;
    size_t port;
};

}

#endif