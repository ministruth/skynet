import TableOp from '@/common_components/layout/table/opBtn';
import TableBtn from '@/common_components/layout/table/tableBtn';
import { API_PREFIX } from '@/config';
import { checkAPI, getIntl, putAPI, UserPerm } from '@/utils';
import { EditOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import _ from 'lodash';
import { AgentBtnProps } from './agentUpdateBtn';
import { PassiveAgentColumns } from './passiveBtn';

const PassiveUpdate: React.FC<AgentBtnProps> = (props) => {
  const intl = getIntl();
  const handleUpdate = async (params: ParamsType) => {
    _.forEach(params, (v, k) => {
      if (_.isEqual(props.initialValues?.[k], v)) delete params[k];
    });
    if (
      await checkAPI(
        putAPI(
          `${API_PREFIX}/passive_agents/${props.initialValues?.id}`,
          params,
        ),
      )
    ) {
      props.tableRef?.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };

  return (
    <TableOp
      title={intl.get('pages.config.agent.op.update.passive.title')}
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
        columns: PassiveAgentColumns(intl),
        initialValues: props.initialValues,
      }}
      width={500}
      changedSubmit={true}
    />
  );
};

export default PassiveUpdate;
