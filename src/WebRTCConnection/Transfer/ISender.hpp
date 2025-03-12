#ifndef I_SENDER_HPP
#define I_SENDER_HPP

#include <cstdint>
#include <vector>

namespace Airlink {

class ISender {
  public:
	ISender() = default;

  public:
	virtual void sendData(std::vector<uint8_t> &) = 0;
	virtual void sendData(std::vector<uint8_t> &&) = 0;
	virtual void sendData(std::vector<uint8_t> &) const = 0;
	virtual void sendData(std::vector<uint8_t> &&) const = 0;
};

} // namespace Airlink

#endif