import { checkAPI, fileToBase64, getAPI, getIntl, putAPI } from '@/utils';
import ProCard from '@ant-design/pro-card';
import {
  ProDescriptions,
  ProDescriptionsItemProps,
} from '@ant-design/pro-components';
import { FormattedMessage } from '@umijs/max';
import { Button, Col, message, Space, Upload } from 'antd';
import { UploadChangeParam, UploadFile } from 'antd/es/upload';
import { Row } from 'antd/lib';
import { useEffect, useState } from 'react';
import GeoIP from '../geoip';
import confirm from '../layout/modal';
import HistoryLink from './historyLink';
import PermissionList from './permission';
import SessionList from './session';
import styles from './style.less';
import WebpushList from './webpush';

const AccountCard = () => {
  const intl = getIntl();
  const [data, setData] = useState<{ [x: string]: any }>({});
  const fetch = async () => {
    const msg = await getAPI(`/users/self`);
    setData(msg.data);
  };
  const upload = async (avatar: string) => {
    if (
      await checkAPI(
        putAPI(`/users/self`, {
          avatar: avatar,
        }),
      )
    )
      fetch();
  };
  const reset = async () => {
    confirm({
      title: intl.get('pages.account.reset.title'),
      content: intl.get('app.confirm'),
      onOk() {
        return new Promise((resolve, reject) => {
          putAPI(`/users/self`, {
            avatar: '',
          }).then((rsp) => {
            if (rsp && rsp.code === 0) {
              fetch();
              resolve(rsp);
            } else {
              reject(rsp);
            }
          });
        });
      },
      intl: intl,
    });
  };
  const imgChange = (e: UploadChangeParam<UploadFile<any>>) => {
    if (e.fileList.length > 0) {
      fileToBase64(e.fileList[0].originFileObj)
        .catch((e) => {
          message.error(`Error: ${e.message}`);
          return null;
        })
        .then((f) => {
          if (f) return upload(f);
        });
    }
  };

  useEffect(() => {
    fetch();
  }, []);

  const columns: ProDescriptionsItemProps[] = [
    {
      title: intl.get('app.table.id'),
      dataIndex: 'id',
      copyable: true,
    },
    {
      title: intl.get('tables.username'),
      dataIndex: 'username',
    },
    {
      title: intl.get('tables.avatar'),
      dataIndex: 'avatar',
      className: styles.itemcenter,
      render: (_, row) => {
        return (
          <Space size="large">
            <Upload
              name="avatar"
              listType="picture-card"
              showUploadList={false}
              accept=".png,.jpg,.jpeg,.webp"
              beforeUpload={(file) => {
                if (
                  !['image/png', 'image/jpeg', 'image/webp'].includes(file.type)
                ) {
                  message.error(
                    intl.get('pages.user.form.avatar.invalid', {
                      file: file.name,
                    }),
                  );
                  return Upload.LIST_IGNORE;
                }
                if (file.size > 1024 * 1024) {
                  message.error(intl.get('app.filesize', { size: '1MB' }));
                  return Upload.LIST_IGNORE;
                }
                return false;
              }}
              onChange={imgChange}
            >
              <img src={row.avatar} alt="avatar" style={{ width: '95%' }} />
            </Upload>
            <Button onClick={reset}>
              <FormattedMessage id="app.reset" />
            </Button>
          </Space>
        );
      },
    },
    {
      title: intl.get('tables.lastip'),
      dataIndex: 'last_ip',
      render: (_, row) => {
        return (
          <Space>
            <GeoIP value={row.last_ip} />
            <HistoryLink />
          </Space>
        );
      },
    },
    {
      title: intl.get('tables.lastlogin'),
      dataIndex: 'last_login',
      valueType: 'dateTime',
    },
    {
      title: intl.get('app.table.createdat'),
      dataIndex: 'created_at',
      valueType: 'dateTime',
    },
    {
      title: intl.get('app.table.updatedat'),
      dataIndex: 'updated_at',
      valueType: 'dateTime',
    },
  ];

  return (
    <ProCard bordered>
      <Row>
        <Col xs={24} md={12}>
          <ProDescriptions
            columns={columns}
            emptyText={'-'}
            dataSource={data}
            column={1}
          />
          <SessionList />
        </Col>
        <Col xs={24} md={12}>
          <PermissionList />
          <WebpushList />
        </Col>
      </Row>
    </ProCard>
  );
};

export default AccountCard;
