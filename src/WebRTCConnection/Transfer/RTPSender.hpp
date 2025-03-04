#ifndef RTP_SENDER_HPP
#define RTP_SENDER_HPP


//==================================
#include <uvgrtp/context.hh>
//==================================

//----------------------------------
#include "ISender.hpp"
//----------------------------------

//==================================
namespace uvgrtp {
    class session;
    class media_stream;
    namespace frame {
        struct rtp_frame;
    }
}
//==================================

class RtpSender : public ISender {
public:
    RtpSender();
    ~RtpSender();

    void sendData(std::vector<uint8_t>& data) final override;
    void sendData(std::vector<uint8_t>&& data) final override;
    void sendData(std::vector<uint8_t>& data) const final override;
    void sendData(std::vector<uint8_t>&& data) const final override;
private:
    uvgrtp::context uvgRtpContext;
    uvgrtp::session* rtpSession = nullptr;
    uvgrtp::media_stream* strm = nullptr;
};


#endif