#ifndef ICE_SERVER_CONFIG_HPP
#define ICE_SERVER_CONFIG_HPP

#include <string>

namespace Airlink {

struct IceServerConfig {
	std::string url;
	std::string login;
	std::string password;
};

} // namespace Airlink

#endif