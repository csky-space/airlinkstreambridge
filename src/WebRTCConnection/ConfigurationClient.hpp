#ifndef CONFIGURATION_CLIENT_HPP
#define CONFIGURATION_CLIENT_HPP

#include <memory>
#include <string>
#include <string_view>

#include <Poco/Net/HTTPSClientSession.h>
#include <Poco/URI.h>

#include "WebrtcConfig.hpp"

namespace Airlink {

class ConfigurationClient {
  public:
	ConfigurationClient(std::string_view configurationServerUrl, std::string_view login, std::string_view password);
	WebrtcConfiguration getConfiguration() const;

  private:
	void requestToLogin();
	void requestToConfig();

	void parseLogin();

	Poco::Net::HTTPSClientSession session;
	Poco::Net::Context::Ptr pContext;

	Poco::URI apiURI;

	std::string accessToken;

	WebrtcConfiguration configuration;

	std::string hostUrl;
	std::string login;
	std::string password;
	std::string queryPrefix = "/api/groundStation";
};

} // namespace Airlink

#endif