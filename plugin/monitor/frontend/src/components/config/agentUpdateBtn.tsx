import { Columns } from '@/common_components/layout/table/column';
import TableOp from '@/common_components/layout/table/opBtn';
import TableBtn from '@/common_components/layout/table/tableBtn';
import { API_PREFIX } from '@/config';
import { checkAPI, getIntl, putAPI, UserPerm } from '@/utils';
import { EditOutlined } from '@ant-design/icons';
import { ActionType } from '@ant-design/pro-components';
import { ParamsType } from '@ant-design/pro-provider';
import { Store } from 'antd/es/form/interface';
import _ from 'lodash';

export interface AgentBtnProps {
  tableRef: React.MutableRefObject<ActionType | undefined>;
  initialValues?: Store;
}

const AgentUpdate: React.FC<AgentBtnProps> = (props) => {
  const intl = getIntl();
  const handleUpdate = async (params: ParamsType) => {
    _.forEach(params, (v, k) => {
      if (_.isEqual(props.initialValues?.[k], v)) delete params[k];
    });
    if (
      await checkAPI(
        putAPI(`${API_PREFIX}/agents/${props.initialValues?.id}`, params),
      )
    ) {
      props.tableRef?.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };
  const columns: Columns = (intl) => [
    {
      title: intl.get('pages.agent.table.name'),
      dataIndex: 'name',
      tooltip: intl.get('pages.config.agent.form.name.tip'),
      fieldProps: {
        maxLength: 32,
      },
      formItemProps: {
        rules: [{ required: true }],
      },
    },
  ];

  return (
    <TableOp
      title={intl.get('pages.config.agent.op.update.title')}
      trigger={
        <TableBtn
          key="update"
          icon={EditOutlined}
          tip={intl.get('app.op.update')}
        />
      }
      rollback={<EditOutlined key="update" />}
      permName="manage.plugin"
      perm={UserPerm.PermWrite}
      schemaProps={{
        onFinish: handleUpdate,
        columns: columns(intl),
        initialValues: props.initialValues,
      }}
      width={500}
      changedSubmit={true}
    />
  );
};

export default AgentUpdate;
