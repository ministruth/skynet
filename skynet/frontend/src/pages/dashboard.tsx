import DashboardCard from '@/components/dashboard/card';
import MainLayout from '@/components/layout';
import { UserPerm } from '@/utils';

const Dashboard = () => {
  return (
    <MainLayout title="titles.dashboard" access="user" perm={UserPerm.PermAll}>
      <DashboardCard />
    </MainLayout>
  );
};

export default Dashboard;
