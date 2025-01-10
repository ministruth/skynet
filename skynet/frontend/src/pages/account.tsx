import AccountCard from '@/components/account/card';
import MainLayout from '@/components/layout';
import MainContainer from '@/components/layout/container';
import { getIntl, UserPerm } from '@/utils';

const Account = () => {
  const intl = getIntl();

  return (
    <MainLayout title="titles.account" access="user" perm={UserPerm.PermAll}>
      <MainContainer
        title={intl.get('menus.account')}
        routes={[
          {
            title: 'menus.account',
          },
        ]}
        content={intl.get('pages.account.content')}
      >
        <AccountCard />
      </MainContainer>
    </MainLayout>
  );
};

export default Account;
