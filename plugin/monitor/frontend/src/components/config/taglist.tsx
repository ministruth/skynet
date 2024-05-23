import { PlusOutlined } from '@ant-design/icons';
import { FormattedMessage } from '@umijs/max';
import { Input, InputRef, Tag, Tooltip } from 'antd';
import { useEffect, useRef, useState } from 'react';

export interface TaglistProps {
  value?: string[];
  onChange?: (v: string[]) => void;
  disabled?: boolean;
}

const newTagStyle: React.CSSProperties = {
  height: 22,
  background: 'white',
  borderStyle: 'dashed',
};

const tagInputStyle: React.CSSProperties = {
  width: 128,
  height: 22,
  marginInlineEnd: 8,
  verticalAlign: 'top',
};

const TagList: React.FC<TaglistProps> = (props) => {
  const [inputVisible, setInputVisible] = useState(false);
  const [inputValue, setInputValue] = useState('');
  const [editInputIndex, setEditInputIndex] = useState(-1);
  const [editInputValue, setEditInputValue] = useState('');
  const inputRef = useRef<InputRef>(null);
  const editInputRef = useRef<InputRef>(null);
  useEffect(() => {
    if (inputVisible) {
      inputRef.current?.focus();
    }
  }, [inputVisible]);

  useEffect(() => {
    editInputRef.current?.focus();
  }, [editInputValue]);

  const showInput = () => {
    setInputVisible(true);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value);
  };

  const handleInputConfirm = () => {
    if (inputValue && !props.value?.includes(inputValue)) {
      props.onChange?.([...(props.value ?? []), inputValue]);
    }
    setInputVisible(false);
    setInputValue('');
  };

  const handleClose = (removedTag: string) => {
    const newTags = props.value?.filter((tag) => tag !== removedTag) ?? [];
    props.onChange?.(newTags);
  };

  const handleEditInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setEditInputValue(e.target.value);
  };

  const handleEditInputConfirm = () => {
    const newTags = [...(props.value ?? [])];
    newTags[editInputIndex] = editInputValue;
    props.onChange?.(newTags);
    setEditInputIndex(-1);
    setEditInputValue('');
  };

  return (
    <>
      {(props.value || []).map((tag, index) => {
        if (editInputIndex === index) {
          return (
            <Input
              ref={editInputRef}
              key={tag}
              size="small"
              style={tagInputStyle}
              value={editInputValue}
              onChange={handleEditInputChange}
              onBlur={handleEditInputConfirm}
              onPressEnter={handleEditInputConfirm}
            />
          );
        }
        const longtag = tag.length > 20;
        const tagElem = (
          <Tag
            key={tag}
            onClose={() => handleClose(tag)}
            closable={!props.disabled}
          >
            <span
              onDoubleClick={(e) => {
                if (!props.disabled) {
                  setEditInputIndex(index);
                  setEditInputValue(tag);
                  e.preventDefault();
                }
              }}
            >
              {longtag ? `${tag.slice(0, 20)}...` : tag}
            </span>
          </Tag>
        );
        return longtag ? (
          <Tooltip title={tag} key={tag}>
            {tagElem}
          </Tooltip>
        ) : (
          tagElem
        );
      })}

      {props.disabled ? (
        <></>
      ) : inputVisible ? (
        <Input
          ref={inputRef}
          type="text"
          size="small"
          style={tagInputStyle}
          value={inputValue}
          onChange={handleInputChange}
          onBlur={handleInputConfirm}
          onPressEnter={handleInputConfirm}
        />
      ) : (
        <Tag style={newTagStyle} icon={<PlusOutlined />} onClick={showInput}>
          <FormattedMessage id="pages.config.setting.shell.new" />
        </Tag>
      )}
    </>
  );
};

export default TagList;
