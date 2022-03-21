import GroupCard from '@/components/group/card';
import MainLayout from '@/components/layout';
import MainContainer from '@/components/layout/container';
import { getIntl, UserPerm } from '@/utils';

const Group = () => {
  const intl = getIntl();
  return (
    <MainLayout
      title="titles.group"
      access="manage.group"
      perm={UserPerm.PermRead}
    >
      <MainContainer
        title={intl.get('menus.user.group')}
        routes={[
          {
            path: '',
            breadcrumbName: 'menus.user',
          },
          {
            path: '/group',
            breadcrumbName: 'menus.user.group',
          },
        ]}
        content={intl.get('pages.group.content')}
      >
        <GroupCard />
      </MainContainer>
    </MainLayout>
  );
};

export default Group;
