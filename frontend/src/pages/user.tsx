import MainLayout from '@/components/layout';
import MainContainer from '@/components/layout/container';
import UserCard from '@/components/user/card';
import { getIntl, UserPerm } from '@/utils';

const User = () => {
  const intl = getIntl();
  return (
    <MainLayout
      title="titles.user"
      access="manage.user"
      perm={UserPerm.PermRead}
    >
      <MainContainer
        title={intl.get('menus.user.user')}
        routes={[
          {
            path: '',
            breadcrumbName: 'menus.user',
          },
          {
            path: '/user',
            breadcrumbName: 'menus.user.user',
          },
        ]}
        content={intl.get('pages.user.content')}
      >
        <UserCard />
      </MainContainer>
    </MainLayout>
  );
};

export default User;
