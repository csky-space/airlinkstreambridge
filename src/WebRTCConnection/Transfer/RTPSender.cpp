#include "RTPSender.hpp"
#include "uvgrtp/util.hh"

#include <cstddef>
#include <uvgrtp/lib.hh>

RtpSender::RtpSender() {
    rtpSession = uvgRtpContext.create_session(std::pair{"127.0.0.1", "127.0.0.1"});
    strm = rtpSession->create_stream(8888, RTP_FORMAT_H265, RCE_SEND_ONLY);
}

RtpSender::~RtpSender() {
    rtpSession->destroy_stream(strm);
    uvgRtpContext.destroy_session(rtpSession);
}

void RtpSender::sendData(std::vector<uint8_t>& data) {
    strm->push_frame(data.data(), data.size(), RTP_NO_FLAGS);
}

void RtpSender::sendData(std::vector<uint8_t>&& data) {
    strm->push_frame(data.data(), data.size(), RTP_NO_FLAGS);
}

void RtpSender::sendData(std::vector<uint8_t>& data) const {
    strm->push_frame(data.data(), data.size(), RTP_NO_FLAGS);
}

void RtpSender::sendData(std::vector<uint8_t>&& data) const {
    strm->push_frame(data.data(), data.size(), RTP_NO_FLAGS);
}