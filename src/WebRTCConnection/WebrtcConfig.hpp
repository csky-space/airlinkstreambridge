#ifndef WEBRTC_CONFIG_HPP
#define WEBRTC_CONFIG_HPP

#include <vector>

#include "IceServerConfig.hpp"

namespace Airlink {

struct WebrtcConfiguration {
	std::vector<IceServerConfig> stunServerConfigs;
	std::vector<IceServerConfig> turnServersConfigs;
	std::string wsUrl;
};

} // namespace Airlink

#endif