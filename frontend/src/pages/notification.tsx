import MainLayout from '@/components/layout';
import MainContainer from '@/components/layout/container';
import NotificationCard from '@/components/notification/card';
import { getIntl, UserPerm } from '@/utils';

const Notification = () => {
  const intl = getIntl();
  return (
    <MainLayout
      title="titles.notification"
      access="manage.notification"
      perm={UserPerm.PermRead}
    >
      <MainContainer
        title={intl.get('menus.notification')}
        routes={[
          {
            path: '/notification',
            breadcrumbName: 'menus.notification',
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
