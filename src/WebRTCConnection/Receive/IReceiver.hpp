#ifndef I_RECEIVER_HPP
#define I_RECEIVER_HPP

#include <cstdint>
#include <functional>
#include <vector>

class IReceiver {
public:
    IReceiver() = default;
    virtual bool isOpened() = 0;
    virtual bool isOpened() const = 0;
    virtual bool isClosed() = 0;
    virtual bool isClosed() const = 0;
    virtual void waitForConnection() = 0;

    virtual void onData(const std::function<void(std::vector<uint8_t>&&)>& onVideoMessageAction) = 0;
    virtual void onUpdate() = 0;
};

#endif