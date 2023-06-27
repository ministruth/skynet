import MainLayout from '@/components/layout';
import MainContainer from '@/components/layout/container';
import NotificationCard from '@/components/notification/card';
import { getIntl, UserPerm } from '@/utils';
import { MenuDataItem } from '@ant-design/pro-layout';

const Notification = () => {
  const intl = getIntl();
  return (
    <MainLayout
      title="titles.notification"
      access="manage.notification"
      perm={UserPerm.PermRead}
      postMenuData={(item: MenuDataItem[]) =>
        item.map((p) => (p.path === '/notification' ? { ...p, badge: 0 } : p))
      }
    >
      <MainContainer
        title={intl.get('menus.notification')}
        routes={[
          {
            title: 'menus.notification',
          },
        ]}
        content={intl.get('pages.notification.content')}
      >
        <NotificationCard />
      </MainContainer>
    </MainLayout>
  );
};

export default Notification;
