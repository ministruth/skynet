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
import { Checkbox, Modal, Space, Tag, Tooltip } from 'antd';
import { CheckboxChangeEvent } from 'antd/es/checkbox';
import _ from 'lodash';
import { useRef, useState } from 'react';
import PermItem from '../group/permItem';
import styles from '../group/style.less';
import LoadBtn from '../layout/loadBtn';
import Table from '../layout/table';
import { IDColumn } from '../layout/table/column';
import TableBtn from '../layout/table/tableBtn';

export interface UserPermBtnProps {
  uid: string;
}

const UserPermBtn: React.FC<UserPermBtnProps> = (props) => {
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
    let diff = [];
    for (const [key, value] of Object.entries(permState)) {
      if (oldPermState[key] != value) diff.push({ id: key, perm: value });
    }
    return checkAPI(putAPI(`/user/${id}/permission`, diff)).then(() =>
      ref.current?.reloadAndRest?.(),
    );
  };
  const handlePermChange = (pid: string, perm: UserPerm) => {
    setPermState({ ...permState, [pid]: perm });
  };
  const request = async (_: ParamsType) => {
    let group = (await getAPI(`/user/${props.uid}/permission`)).data;
    let all = (await getAPI('/permission')).data;
    group = group.map((v: any) => {
      if (v.origin)
        v.origin = v.origin.map((e: any) => {
          let permstr = '';
          if (e.perm & UserPerm.PermRead) permstr += 'R';
          if (e.perm & UserPerm.PermWrite) permstr += 'W';
          if (e.perm & UserPerm.PermExecute) permstr += 'X';
          if (permstr === 'RWX') permstr = 'A';
          else if (permstr === '') permstr = 'N';
          return {
            id: e.id,
            name: e.name + ':' + permstr,
          };
        });
      else
        v.origin = [{ id: '0', name: intl.get('pages.permission.table.self') }];
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
    group.forEach((v: any) => {
      data[v.id].perm = v.perm;
      data[v.id].origin = v.origin;
    });
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
      title: intl.get('pages.permission.table.origin'),
      dataIndex: 'origin',
      align: 'center',
      width: 100,
      render: (_, row) => {
        if (!row.origin) return '-';
        return (
          <Space size={0}>
            {row.origin.map((e: any) => (
              <Tooltip key={e.id} title={e.id}>
                <Tag
                  color={e.id === '0' ? 'orange' : undefined}
                  style={{ marginLeft: 2, marginRight: 2 }}
                >
                  {e.name}
                </Tag>
              </Tooltip>
            ))}
          </Space>
        );
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
            basePerm={row.perm}
            perm={permState[row.id]}
            onChange={(perm) => handlePermChange(row.id, perm)}
          />
        );
      },
    },
    {
      title: intl.get('pages.permission.table.inherit'),
      valueType: 'checkbox',
      align: 'center',
      width: 70,
      render: (_, row) => {
        return (
          <Checkbox
            checked={permState[row.id] === UserPerm.PermInherit}
            onChange={(e: CheckboxChangeEvent) => {
              if (e.target.checked)
                setPermState({ ...permState, [row.id]: UserPerm.PermInherit });
              else setPermState({ ...permState, [row.id]: UserPerm.PermNone });
            }}
            className={
              permState[row.id] === UserPerm.PermInherit
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
          tip={intl.get('pages.user.op.perm.tip')}
          onClick={() => setIsModalOpen(true)}
        />
        <Modal
          title={intl.get('pages.user.op.perm.title')}
          open={isModalOpen}
          footer={null}
          onCancel={() => {
            setIsModalOpen(false);
          }}
          width={1000}
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
            request={request}
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
                    !checkPerm(
                      initialState?.signin,
                      access,
                      'manage.user',
                      UserPerm.PermWriteExecute,
                    ) || _.isEqual(permState, oldPermState)
                  }
                  onClick={() => handleUpdate(props.uid)}
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

export default UserPermBtn;
