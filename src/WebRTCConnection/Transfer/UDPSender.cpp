#include "UDPSender.hpp"


UDPSender::UDPSender(QHostAddress address, size_t port) 
    : ISender()
    , address(address)
    , port(port)
{
    //socket.bind(address, port);
}

void UDPSender::sendData(std::vector<uint8_t>& data) {
    socket.writeDatagram(QByteArray(reinterpret_cast<const char*>(data.data()), data.size()), address, port);
}

void UDPSender::sendData(std::vector<uint8_t>&& data) {
    socket.writeDatagram(QByteArray(reinterpret_cast<const char*>(data.data()), data.size()), address, port);
    socket.waitForBytesWritten(100);
}

void UDPSender::sendData(std::vector<uint8_t>& data) const {
    //socket.writeDatagram(QByteArray(reinterpret_cast<const char*>(data.data()), data.size()), address, port);
}

void UDPSender::sendData(std::vector<uint8_t>&& data) const {
    //socket.writeDatagram(QByteArray(reinterpret_cast<const char*>(data.data()), data.size()), address, port);
}