self.addEventListener('push', (e) => {
  const data = e.data.json();
  e.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: data.icons,
      data: {
        url: data.url,
      },
    }),
  );
});

self.addEventListener('notificationclick', async (e) => {
  let url = e.notification.data.url;
  e.notification.close();
  e.waitUntil(
    self.clients
      .matchAll({
        type: 'window',
        includeUncontrolled: true,
      })
      .then((clients) => {
        if (clients.length > 0) {
          const client = clients[0];
          return client.postMessage({
            action: 'skynet-click',
            url: url,
          });
        } else return clients.openWindow(url);
      }),
  );
});
