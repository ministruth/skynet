import {
  checkAPI,
  checkPerm,
  getAPI,
  getIntl,
  putAPI,
  UserPerm,
} from '@/utils';
import { ProfileOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { useAccess, useModel } from '@umijs/max';
import { Checkbox, Modal } from 'antd';
import { CheckboxChangeEvent } from 'antd/es/checkbox';
import _ from 'lodash';
import { useRef, useState } from 'react';
import LoadBtn from '../layout/loadBtn';
import Table from '../layout/table';
import { IDColumn } from '../layout/table/column';
import TableBtn from '../layout/table/tableBtn';
import PermItem from './permItem';
import styles from './style.less';

export interface GroupPermProps {
  gid: string;
  disableModify?: boolean;
}

const request = async (id: string, _: ParamsType) => {
  let group = (await getAPI(`/group/${id}/permission`)).data;
  let all = (await getAPI('/permission')).data;
  group = group.map((v: any) => {
    if (v.perm === UserPerm.PermNone) v.perm = UserPerm.PermBan;
    return v;
  });
  all = all.map((v: any) => {
    v.perm = UserPerm.PermNone;
    return v;
  });
  let data = all.reduce((tot: any, v: any) => {
    tot[v.id] = v;
    return tot;
  }, {});
  group.forEach((v: any) => (data[v.id].perm = v.perm));
  const perm: any = Object.values(data);
  perm.sort((a: any, b: any) => {
    if (a.name > b.name) return 1;
    if (a.name < b.name) return -1;
    return 0;
  });

  return {
    data: perm,
    success: true,
  };
};

const GroupPerm: React.FC<GroupPermProps> = (props) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const intl = getIntl();
  const { initialState } = useModel('@@initialState');
  const access = useAccess();
  const ref = useRef<ActionType>();
  const [permState, setPermState] = useState<{ [Key: string]: UserPerm }>({});
  const [oldPermState, setOldPermState] = useState<{ [Key: string]: UserPerm }>(
    {},
  );
  const handleUpdate = (id: string) => {
    const rotate = (v: UserPerm) => {
      if (v !== UserPerm.PermNone && v !== UserPerm.PermBan) return v;
      if (v === UserPerm.PermBan) return 0;
      if (v === UserPerm.PermNone) return -1;
    };
    let diff = [];
    for (const [key, value] of Object.entries(permState)) {
      if (oldPermState[key] != value)
        diff.push({ id: key, perm: rotate(value) });
    }
    return checkAPI(putAPI(`/group/${id}/permission`, diff)).then(() =>
      ref.current?.reloadAndRest?.(),
    );
  };
  const handlePermChange = (pid: string, perm: UserPerm) => {
    setPermState({ ...permState, [pid]: perm });
  };
  const columns: ProColumns[] = [
    IDColumn(intl),
    {
      title: intl.get('pages.permission.table.name'),
      dataIndex: 'name',
      align: 'center',
      ellipsis: true,
      onCell: () => {
        return {
          style: {
            maxWidth: 150,
          },
        };
      },
    },
    {
      title: intl.get('pages.permission.table.note'),
      dataIndex: 'note',
      ellipsis: true,
      align: 'center',
      onCell: () => {
        return {
          style: {
            maxWidth: 150,
          },
        };
      },
    },
    {
      title: intl.get('pages.permission.table.perm'),
      valueType: 'checkbox',
      align: 'center',
      width: 200,
      render: (_, row) => {
        return (
          <PermItem
            disabled={props.disableModify}
            basePerm={row.perm}
            perm={permState[row.id]}
            onChange={(perm) => handlePermChange(row.id, perm)}
          />
        );
      },
    },
    {
      title: intl.get('pages.permission.table.ban'),
      valueType: 'checkbox',
      align: 'center',
      width: 50,
      render: (_, row) => {
        return (
          <Checkbox
            disabled={props.disableModify}
            checked={permState[row.id] === UserPerm.PermBan}
            onChange={(e: CheckboxChangeEvent) => {
              if (e.target.checked)
                setPermState({ ...permState, [row.id]: UserPerm.PermBan });
              else setPermState({ ...permState, [row.id]: UserPerm.PermNone });
            }}
            className={
              (permState[row.id] === UserPerm.PermBan &&
                oldPermState[row.id] !== UserPerm.PermBan) ||
              (permState[row.id] !== UserPerm.PermBan &&
                oldPermState[row.id] === UserPerm.PermBan)
                ? styles['checkbox-orange']
                : undefined
            }
          />
        );
      },
    },
  ];

  if (
    !checkPerm(initialState?.signin, access, 'manage.user', UserPerm.PermRead)
  )
    return <ProfileOutlined key="perm" />;
  else
    return (
      <>
        <TableBtn
          icon={ProfileOutlined}
          tip={intl.get('pages.group.op.perm.tip')}
          onClick={() => setIsModalOpen(true)}
        />
        <Modal
          title={intl.get('pages.group.op.perm.title')}
          open={isModalOpen}
          footer={null}
          onCancel={() => {
            setIsModalOpen(false);
          }}
          width={900}
          destroyOnClose={true}
        >
          <Table
            postData={(data: any) => {
              let stat: { [Key: string]: UserPerm } = {};
              data.forEach((v: any) => {
                stat[v.id] = v.perm;
              });
              setPermState(stat);
              setOldPermState(stat);
              return data;
            }}
            className={styles.scrolltable}
            scroll={{ x: 'max-content', y: 400 }}
            search={false}
            headerTitle={undefined}
            actionRef={ref}
            rowKey="id"
            request={(params) => request(props.gid, params)}
            columns={columns}
            cardBordered={false}
            cardProps={{ bodyStyle: { paddingInline: 0, paddingBlock: 0 } }}
            pagination={false}
            toolbar={{
              actions: [
                <LoadBtn
                  key="update"
                  type="primary"
                  disabled={
                    props.disableModify ||
                    !checkPerm(
                      initialState?.signin,
                      access,
                      'manage.user',
                      UserPerm.PermWriteExecute,
                    ) ||
                    _.isEqual(permState, oldPermState)
                  }
                  onClick={() => handleUpdate(props.gid)}
                >
                  {intl.get('pages.permission.op.update')}
                </LoadBtn>,
              ],
            }}
          />
        </Modal>
      </>
    );
};

export default GroupPerm;
