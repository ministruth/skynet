import { checkAPI, getIntl, putAPI } from '@/utils';
import { ParamsType } from '@ant-design/pro-provider';
import { FormattedMessage } from '@umijs/max';
import { Button } from 'antd';
import { Store } from 'antd/es/form/interface';
import _ from 'lodash';
import ModalSchema from '../layout/modalSchema';
import { AvatarColumn } from '../user/card';

export interface UpdateBtnProps {
  initialValues: Store;
  reload: () => void;
}

const UpdateBtn: React.FC<UpdateBtnProps> = (props) => {
  const intl = getIntl();
  const columns = [
    {
      title: intl.get('tables.username'),
      dataIndex: 'username',
      tooltip: intl.get('pages.user.form.username.tip'),
      readonly: true,
    },
    {
      title: intl.get('tables.password'),
      dataIndex: 'password',
      valueType: 'password',
      fieldProps: {
        placeholder: intl.get('pages.user.form.password.placeholder'),
      },
    },
    AvatarColumn(intl),
  ];
  const handleUpdate = async (params: ParamsType) => {
    _.forEach(params, (v, k) => {
      if (_.isEqual(props.initialValues[k], v)) delete params[k];
    });

    if (await checkAPI(putAPI(`/users/self`, params))) {
      props.reload();
      return true;
    }
    return false;
  };

  return (
    <ModalSchema
      title={intl.get('pages.user.update.title')}
      trigger={
        <Button size="small">
          <FormattedMessage id="pages.dashboard.update" />
        </Button>
      }
      width={500}
      schemaProps={{
        layout: 'horizontal',
        autoFocusFirstInput: true,
        labelCol: { span: 6 },
        request: async (_params: Record<string, any>, _props: any) => {
          props.initialValues!.password = '';
          return props.initialValues!;
        },
        onFinish: handleUpdate,
        columns: columns as any,
        initialValues: props.initialValues,
      }}
      changedSubmit={true}
    />
  );
};

export default UpdateBtn;
