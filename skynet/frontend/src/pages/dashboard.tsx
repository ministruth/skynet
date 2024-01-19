import MainLayout from '@/components/layout';
import { UserPerm } from '@/utils';

const Dashboard = () => {
  return (
    <MainLayout
      title="titles.dashboard"
      access="user"
      perm={UserPerm.PermAll}
    ></MainLayout>
  );
};

export default Dashboard;
