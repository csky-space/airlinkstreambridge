#ifndef I_RECEIVER_HPP
#define I_RECEIVER_HPP

class IReceiver {
public:
    IReceiver() = default;
    virtual bool isOpened() = 0;
    virtual bool isOpened() const = 0;
    virtual bool isClosed() = 0;
    virtual bool isClosed() const = 0;
};

#endif