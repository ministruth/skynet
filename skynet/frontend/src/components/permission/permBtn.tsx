import {
  checkAPI,
  checkPerm,
  getAPI,
  getIntl,
  putAPI,
  UserPerm,
} from '@/utils';
import { ProfileOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-components';
import { ActionType, ProColumns } from '@ant-design/pro-table';
import { useAccess } from '@umijs/max';
import { Checkbox, Modal, Space, Tag, Tooltip } from 'antd';
import { CheckboxChangeEvent } from 'antd/es/checkbox';
import _ from 'lodash';
import { useRef, useState } from 'react';
import LoadBtn from '../layout/loadBtn';
import Table from '../layout/table';
import { IDColumn } from '../layout/table/column';
import TableBtn from '../layout/table/tableBtn';
import PermItem from './permItem';
import styles from './style.less';

export interface PermProps {
  ugid: string;
  origin: boolean;
  disabled?: boolean;
  disableModify?: boolean;
  refresh?: boolean;
}

const Permission: React.FC<PermProps> = (props) => {
  const base = props.origin ? 'users' : 'groups';
  const [isModalOpen, setIsModalOpen] = useState(false);
  const intl = getIntl();
  const access = useAccess();
  const ref = useRef<ActionType>();
  const [permState, setPermState] = useState<{ [Key: string]: UserPerm }>({});
  const [oldPermState, setOldPermState] = useState<{ [Key: string]: UserPerm }>(
    {},
  );
  const perm_disabled = !checkPerm(access, 'manage.user', UserPerm.PermWrite);
  const request = async (_: ParamsType) => {
    let obj = (await getAPI(`/${base}/${props.ugid}/permissions`)).data;
    let all = (await getAPI('/permissions')).data;
    obj = obj.map((v: any) => {
      if (v.perm === UserPerm.PermNone) v.perm = UserPerm.PermBan;
      if (props.origin) {
        if (v.origin)
          v.origin = v.origin.map((e: any) => {
            let permstr = '';
            if (e.perm & UserPerm.PermRead) permstr += 'R';
            if (e.perm & UserPerm.PermWrite) permstr += 'W';
            if (permstr === 'RW') permstr = 'A';
            else if (permstr === '') permstr = 'N';
            return {
              id: e.id,
              name: e.name + ':' + permstr,
            };
          });
        else v.origin = [{ id: '0', name: intl.get('tables.self') }];
      }
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
    obj.forEach((v: any) => {
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
  const handleUpdate = () => {
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
    return checkAPI(putAPI(`/${base}/${props.ugid}/permissions`, diff)).then(
      (rsp) => {
        if (rsp) {
          ref.current?.reloadAndRest?.();
          if (props.refresh) window.location.reload();
        }
      },
    );
  };
  const handlePermChange = (pid: string, perm: UserPerm) => {
    setPermState({ ...permState, [pid]: perm });
  };
  let columns: ProColumns[] = [
    IDColumn(intl),
    {
      title: intl.get('tables.name'),
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
      title: intl.get('tables.note'),
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
  ];
  if (props.origin)
    columns = columns.concat([
      {
        title: intl.get('tables.origin'),
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
    ]);
  columns = columns.concat([
    {
      title: intl.get('tables.perm'),
      valueType: 'checkbox',
      align: 'center',
      width: 180,
      render: (_, row) => {
        return (
          <PermItem
            disabled={props.disableModify || perm_disabled}
            basePerm={row.perm}
            perm={permState[row.id]}
            onChange={(perm) => handlePermChange(row.id, perm)}
          />
        );
      },
    },
    {
      title: intl.get('tables.ban'),
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
  ]);

  if (!checkPerm(access, 'manage.user', UserPerm.PermRead) || props.disabled)
    return <ProfileOutlined key="perm" />;
  else
    return (
      <>
        <TableBtn
          icon={ProfileOutlined}
          tip={intl.get('pages.permission.tip')}
          onClick={() => setIsModalOpen(true)}
        />
        <Modal
          title={intl.get(`pages.permission.${base.slice(0, -1)}.title`)}
          open={isModalOpen}
          footer={null}
          onCancel={() => {
            setIsModalOpen(false);
          }}
          width={props.origin ? 1000 : 900}
          destroyOnClose={true}
          maskClosable={false}
        >
          <Table
            postData={(data: any) => {
              let stat: { [Key: string]: UserPerm } = {};
              data.forEach((v: any) => {
                stat[v.id] = v.perm;
              });
              data = _.sortBy(data, 'updated_at');
              setPermState(stat);
              setOldPermState(stat);
              return data;
            }}
            className={styles.scrolltable}
            scroll={{ x: 'auto', y: 400 }}
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
                    props.disableModify ||
                    !checkPerm(access, 'manage.user', UserPerm.PermWrite) ||
                    _.isEqual(permState, oldPermState)
                  }
                  onClick={handleUpdate}
                >
                  {intl.get('app.op.update')}
                </LoadBtn>,
              ],
            }}
          />
        </Modal>
      </>
    );
};

export default Permission;
