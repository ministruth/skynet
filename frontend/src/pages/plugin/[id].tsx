import MainLayout from '@/components/layout';
import { getIntl } from '@/utils';
import { MicroApp, useParams } from 'umi';

const PluginChild = () => {
  const intl = getIntl();
  const { id } = useParams<{ id: string }>();
  return (
    <MainLayout>
      <MicroApp name={id} />
    </MainLayout>
  );
};

PluginChild.exact = false; // match all subpath

export default PluginChild;
