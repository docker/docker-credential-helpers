#include <Security/Security.h>

struct Server {
  SecProtocolType proto;
  char *host;
  char *path;
  unsigned int port;
};

char *keychain_add(struct Server *server, char *username, char *password);
char *keychain_get(struct Server *server, unsigned int *username_l, char **username, unsigned int *password_l, char **password);
char *keychain_delete(struct Server *server);
