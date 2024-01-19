import MainLayout from '@/components/layout';
import { MicroApp, useAccess, useModel } from '@umijs/max';
import { useParams } from 'umi';

const PluginChild = () => {
  const { id } = useParams<{ id: string }>();
  const { initialState } = useModel('@@initialState');
  const access = useAccess();

  return (
    <MainLayout>
      <MicroApp
        name={id}
        base={`/plugin/${id}`}
        autoSetLoading
        initialState={initialState}
        access={access}
      />
    </MainLayout>
  );
};

PluginChild.exact = false; // match all subpath

export default PluginChild;
